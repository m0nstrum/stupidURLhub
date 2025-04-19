package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"paste-service/internal/clients/sluggen"
	"paste-service/internal/clients/tagger"
	"paste-service/internal/model"
	"paste-service/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidPaste             = errors.New("некорректные данные пасты")
	ErrPasteNotFound            = errors.New("паста не найдена")
	ErrInvalidEditToken         = errors.New("неверный токен редактирования")
	ErrPasteExpired             = errors.New("срок действия пасты истек")
	ErrTaggerUnavailable        = errors.New("сервис тэггера недоступен")
	ErrSlugGeneratorUnavailable = errors.New("сервис генерации slug недоступен")
)

type CreatePasteRequest struct {
	Content   string         `json:"content"`
	Tags      []string       `json:"tags,omitempty"`
	ExpiresIn *time.Duration `json:"expires_in,omitempty"`
	AutoTag   bool           `json:"auto_tag"`
}

type PasteResponse struct {
	ID         string     `json:"id"`
	Slug       string     `json:"slug"`
	Content    string     `json:"content"`
	Tags       []string   `json:"tags"`
	ViewCount  int        `json:"view_count"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastViewed *time.Time `json:"last_viewed,omitempty"`
	Expires    *time.Time `json:"expires,omitempty"`
}

type EditResponse struct {
	PasteResponse
	EditToken string `json:"edit_token"`
}

type PasteService struct {
	repo       *repository.PasteRepository
	tagger     tagger.TaggerClient
	sluggen    sluggen.SlugClient
	maxTagsLen int
}

func NewPasteService(
	repo *repository.PasteRepository,
	tagger tagger.TaggerClient,
	sluggen sluggen.SlugClient,
) *PasteService {
	return &PasteService{
		repo:       repo,
		tagger:     tagger,
		sluggen:    sluggen,
		maxTagsLen: 10,
	}
}

func generateEditToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func verifyToken(token, hash string) bool {
	tokenHash := hashToken(token)
	return tokenHash == hash
}

func (s *PasteService) convertPasteToResponse(paste *model.Paste) PasteResponse {
	return PasteResponse{
		ID:         paste.ID,
		Slug:       paste.Slug,
		Content:    paste.Content,
		Tags:       paste.Tags,
		ViewCount:  paste.ViewCount,
		CreatedAt:  paste.CreatedAt,
		UpdatedAt:  paste.UpdatedAt,
		LastViewed: paste.LastViewed,
		Expires:    paste.Expires,
	}
}

func (s *PasteService) CreatePaste(req CreatePasteRequest) (*EditResponse, error) {
	if req.Content == "" {
		return nil, ErrInvalidPaste
	}
	v7Uuid, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации UUID v7: %v", err)
	}
	id := v7Uuid.String()

	tags := req.Tags
	if len(tags) == 0 && req.AutoTag {
		var err error
		tags, err = s.tagger.GetTags(req.Content)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTaggerUnavailable, err)
		}
	}

	if len(tags) > s.maxTagsLen {
		tags = tags[:s.maxTagsLen]
	}

	slug, err := s.sluggen.GenerateSlug(req.Content, tags)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSlugGeneratorUnavailable, err)
	}

	editToken, err := generateEditToken()
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации токена: %v", err)
	}

	hashedToken := hashToken(editToken)

	now := time.Now()
	paste := &model.Paste{
		ID:        id,
		Slug:      slug,
		Content:   req.Content,
		EditToken: hashedToken,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.ExpiresIn != nil {
		expiresAt := now.Add(*req.ExpiresIn)
		paste.Expires = &expiresAt
	}

	if err := s.repo.CreatePaste(paste); err != nil {
		return nil, err
	}

	return &EditResponse{
		PasteResponse: s.convertPasteToResponse(paste),
		EditToken:     editToken,
	}, nil
}

func (s *PasteService) GetPaste(slug string) (*PasteResponse, error) {
	paste, err := s.repo.GetPasteBySlug(slug)
	if err != nil {
		if errors.Is(err, repository.ErrPasteNotFound) {
			return nil, ErrPasteNotFound
		}
		if errors.Is(err, repository.ErrPasteExpired) {
			return nil, ErrPasteExpired
		}
		return nil, err
	}

	if err := s.repo.IncrementViewCount(slug); err != nil {
		fmt.Printf("Ошибка при инкрементировании счетчика просмотров: %v\n", err)
	} else {
		updatedPaste, err := s.repo.GetPasteBySlug(slug)
		if err == nil {
			paste = updatedPaste
		} else {
			now := time.Now()
			paste.ViewCount++
			if paste.LastViewed == nil {
				paste.LastViewed = &now
			} else {
				*paste.LastViewed = now
			}
		}
	}

	response := s.convertPasteToResponse(paste)
	return &response, nil
}

func (s *PasteService) UpdatePaste(slug, editToken, content string, tags []string) (*PasteResponse, error) {
	paste, err := s.repo.GetPasteBySlug(slug)
	if err != nil {
		if errors.Is(err, repository.ErrPasteNotFound) {
			return nil, ErrPasteNotFound
		}
		if errors.Is(err, repository.ErrPasteExpired) {
			return nil, ErrPasteExpired
		}
		return nil, err
	}

	if !verifyToken(editToken, paste.EditToken) {
		return nil, ErrInvalidEditToken
	}

	paste.Content = content

	if tags != nil {
		if len(tags) > s.maxTagsLen {
			tags = tags[:s.maxTagsLen]
		}
		paste.Tags = tags
	}

	if err := s.repo.UpdatePaste(paste); err != nil {
		return nil, err
	}

	response := s.convertPasteToResponse(paste)
	return &response, nil
}

func (s *PasteService) GetTopPastes(limit int) ([]PasteResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	pastes, err := s.repo.GetTopPastes(limit)
	if err != nil {
		return nil, err
	}

	result := make([]PasteResponse, len(pastes))
	for i, paste := range pastes {
		result[i] = s.convertPasteToResponse(&paste)
	}

	return result, nil
}

func (s *PasteService) GetRecentPastes(limit int) ([]PasteResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	pastes, err := s.repo.GetRecentPastes(limit)
	if err != nil {
		return nil, err
	}

	result := make([]PasteResponse, len(pastes))
	for i, paste := range pastes {
		result[i] = s.convertPasteToResponse(&paste)
	}

	return result, nil
}
