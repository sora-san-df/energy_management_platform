package anomaly

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidFilter = errors.New("invalid anomaly filter")
	ErrInvalidStatus = errors.New("invalid anomaly status")
)

type Repository interface {
	List(ctx context.Context, f Filter) ([]Anomaly, error)
	Get(ctx context.Context, id int64) (Anomaly, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
}

type ListQuery struct {
	MeterID  string
	Type     Type
	Severity Severity
	Status   Status
}

func (q ListQuery) validate() error {
	if q.Type != "" && q.Type.Weight() == 0 {
		return fmt.Errorf("%w: type %q", ErrInvalidFilter, q.Type)
	}
	if q.Severity != "" && q.Severity.Weight() == 0 {
		return fmt.Errorf("%w: severity %q", ErrInvalidFilter, q.Severity)
	}
	if q.Status != "" && !q.Status.IsValid() {
		return fmt.Errorf("%w: status %q", ErrInvalidFilter, q.Status)
	}
	return nil
}

func (q ListQuery) matches(a Anomaly) bool {
	return (q.MeterID == "" || a.MeterID == q.MeterID) &&
		(q.Type == "" || a.Type == q.Type) &&
		(q.Severity == "" || a.Severity == q.Severity) &&
		(q.Status == "" || a.Status == q.Status)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ranked(ctx context.Context) ([]Anomaly, error) {
	all, err := s.repo.List(ctx, Filter{})
	if err != nil {
		return nil, err
	}
	return Rank(all), nil
}

func (s *Service) List(ctx context.Context, q ListQuery) ([]Anomaly, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}
	ranked, err := s.ranked(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Anomaly, 0, len(ranked))
	for _, a := range ranked {
		if q.matches(a) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Anomaly, error) {
	ranked, err := s.ranked(ctx)
	if err != nil {
		return Anomaly{}, err
	}
	for _, a := range ranked {
		if a.ID == id {
			return a, nil
		}
	}
	return Anomaly{}, ErrNotFound
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, status Status) (Anomaly, error) {
	if !status.IsValid() {
		return Anomaly{}, fmt.Errorf("%w: %q", ErrInvalidStatus, status)
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return Anomaly{}, err
	}
	return s.Get(ctx, id)
}
