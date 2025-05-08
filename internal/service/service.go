package service

import (
	"context"
	"sync"

	"github.com/Jonnydevp/subpub/subpub"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/Jonnydevp/subpub/proto"
)

type PubSubService struct {
	pb.UnimplementedPubSubServer
	bus    subpub.SubPub
	logger *zap.Logger
	mu     sync.Mutex
	subs   map[string]map[subpub.Subscription]struct{}
}

func New(bus subpub.SubPub, logger *zap.Logger) *PubSubService {
	return &PubSubService{
		bus:    bus,
		logger: logger,
		subs:   make(map[string]map[subpub.Subscription]struct{}),
	}
}

func (s *PubSubService) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	key := req.GetKey()
	if key == "" {
		return status.Error(codes.InvalidArgument, "key cannot be empty")
	}

	s.logger.Info("New subscription", zap.String("key", key))

	msgChan := make(chan string, 100)
	defer close(msgChan)

	sub, err := s.bus.Subscribe(key, func(msg interface{}) {
		if str, ok := msg.(string); ok {
			select {
			case msgChan <- str:
			default:
				s.logger.Warn("Message channel full", zap.String("key", key))
			}
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}

	s.mu.Lock()
	if _, ok := s.subs[key]; !ok {
		s.subs[key] = make(map[subpub.Subscription]struct{})
	}
	s.subs[key][sub] = struct{}{}
	s.mu.Unlock()

	defer func() {
		sub.Unsubscribe()
		s.mu.Lock()
		delete(s.subs[key], sub)
		if len(s.subs[key]) == 0 {
			delete(s.subs, key)
		}
		s.mu.Unlock()
	}()

	for {
		select {
		case msg := <-msgChan:
			if err := stream.Send(&pb.Event{Data: msg}); err != nil {
				s.logger.Error("Failed to send event", zap.Error(err))
				return err
			}
		case <-stream.Context().Done():
			s.logger.Info("Subscription closed by client", zap.String("key", key))
			return nil
		}
	}
}

func (s *PubSubService) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	key := req.GetKey()
	data := req.GetData()

	if key == "" {
		return nil, status.Error(codes.InvalidArgument, "key cannot be empty")
	}

	if err := s.bus.Publish(key, data); err != nil {
		s.logger.Error("Failed to publish", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to publish: %v", err)
	}

	s.logger.Debug("Message published", zap.String("key", key), zap.String("data", data))
	return &emptypb.Empty{}, nil
}
