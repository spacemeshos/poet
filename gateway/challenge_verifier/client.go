package challenge_verifier

import (
	"context"

	pb "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/spacemeshos/smutil/log"
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
	logger := logging.FromContext(ctx).WithFields(log.String("gateway", c.gateway))

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
		logger.With().Debug("challenge is invalid", log.String("message", st.Message()))
		return nil, ErrChallengeInvalid
	default:
		logger.With().Debug("could not verify challenge", log.String("message", st.Message()))
		return nil, ErrCouldNotVerify
	}
}

func NewClient(conn *grpc.ClientConn) Verifier {
	return &grpcChallengeVerifierClient{
		client:  pb.NewGatewayServiceClient(conn),
		gateway: conn.Target(),
	}
}
