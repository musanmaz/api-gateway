package store

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"gateway-control/internal/model"
	"gateway-control/internal/validate"
)

var nodeNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type HeartbeatInput struct {
    TrafficBootID string
    Traffic []model.ServiceTrafficBucket
    Telemetry model.NodeTelemetry
	Node              string
	DeliveredRevision *int64
	AckedRevision     *int64
	NackRevision      *int64
	NackDetail        string
	ClearNack         bool
	Probes            []model.Probe
}

func pinDesired(ctx context.Context, tx pgx.Tx, fleetID, node string, revision int64) error {
	if node == "" {
		_, err := tx.Exec(ctx, `
			UPDATE nodes
			SET desired_revision=$2, nack_revision=NULL, nack_detail=''
			WHERE fleet_id=$1`, fleetID, revision)
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE nodes
		SET desired_revision=$3, nack_revision=NULL, nack_detail=''
		WHERE fleet_id=$1 AND name=$2`, fleetID, node, revision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func normalizePublishNode(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if !nodeNamePattern.MatchString(name) {
		return "", &validate.Error{Details: []string{"düğüm adı geçersiz"}}
	}
	return name, nil
}
