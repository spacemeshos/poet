package challenge_verifier

import (
	"context"

	v1 "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/spacemeshos/smutil/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/types"
)

type grpcChallengeVerifierClient struct {
	client  v1.GatewayServiceClient
	gateway string
}

func (c *grpcChallengeVerifierClient) Verify(ctx context.Context, challenge, signature []byte) (*types.ChallengeVerificationResult, error) {
	logger := logging.FromContext(ctx).WithFields(log.String("gateway", c.gateway))

	resp, err := c.client.VerifyChallenge(ctx, &v1.VerifyChallengeRequest{
		Challenge: challenge,
		Signature: signature,
	})

	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.OK:
		return &types.ChallengeVerificationResult{
			Hash:   resp.Hash,
			NodeId: resp.NodeId,
		}, nil
	case codes.InvalidArgument:
		logger.With().Debug("challenge is invalid", log.String("message", st.Message()))
		return nil, types.ErrChallengeInvalid
	default:
		logger.With().Debug("could not verify challenge", log.String("message", st.Message()))
		return nil, types.ErrCouldNotVerify
	}
}

func NewClient(conn *grpc.ClientConn) types.ChallengeVerifier {
	return &grpcChallengeVerifierClient{
		client:  v1.NewGatewayServiceClient(conn),
		gateway: conn.Target(),
	}
}
