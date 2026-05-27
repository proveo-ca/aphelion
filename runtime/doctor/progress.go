//go:build linux

package doctor

import "context"

type Progress interface {
	Surface(ctx context.Context, text string)
}

func surfaceDoctorProgress(ctx context.Context, progress Progress, text string) {
	if progress == nil {
		return
	}
	progress.Surface(ctx, text)
}
