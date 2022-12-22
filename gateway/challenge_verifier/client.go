package challenge_verifier

import (
	"context"

	pb "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spacemeshos/poet/logging"
)

type grpcChallengeVerifierClient struct {
	client  pb.GatewayServiceClient
	gateway string
}

func (c *grpcChallengeVerifierClient) Verify(ctx context.Context, challenge, signature []byte) (*Result, error) {
	logger := logging.FromContext(ctx).With(zap.String("gateway", c.gateway))

	resp, err := c.client.VerifyChallenge(ctx, &pb.VerifyChallengeRequest{
		Challenge: challenge,
		Signature: signature,
	})

	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.OK:
		return &Result{
			Hash:   resp.Hash,
			NodeId: resp.NodeId,
		}, nil
	case codes.InvalidArgument:
		logger.Debug("challenge is invalid", zap.String("message", st.Message()))
		return nil, ErrChallengeInvalid
	default:
		logger.Debug("could not verify challenge", zap.String("message", st.Message()))
		return nil, ErrCouldNotVerify
	}
}

func NewClient(conn *grpc.ClientConn) Verifier {
	return &grpcChallengeVerifierClient{
		client:  pb.NewGatewayServiceClient(conn),
		gateway: conn.Target(),
	}
}
