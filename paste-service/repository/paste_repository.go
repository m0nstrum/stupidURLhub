package repository

import (
	"errors"
	"time"

	"paste-service/internal/cache"
	"paste-service/internal/model"

	"gorm.io/gorm"
)

var (
	ErrPasteNotFound = errors.New("паста не найдена")
	ErrPasteExpired  = errors.New("срок действия пасты истек")
)

type PasteRepository struct {
	DB       *gorm.DB
	Cache    cache.Cache
	cacheTTL time.Duration
}

func NewPasteRepository(db *gorm.DB, cache cache.Cache, cacheTTL time.Duration) *PasteRepository {
	return &PasteRepository{
		DB:       db,
		Cache:    cache,
		cacheTTL: cacheTTL,
	}
}

func (r *PasteRepository) CreatePaste(p *model.Paste) error {
	if err := p.Validate(); err != nil {
		return err
	}

	err := r.DB.Create(p).Error
	if err != nil {
		return err
	}
	r.Cache.Set(p.Slug, p, r.cacheTTL)
	return nil
}

func (r *PasteRepository) GetPasteBySlug(slug string) (*model.Paste, error) {
	// типизированное
	var cachedPaste model.Paste
	if r.Cache.GetTyped(slug, &cachedPaste) {
		if cachedPaste.HasExpired() {
			r.Cache.Invalidate(slug)
			return nil, ErrPasteExpired
		}
		return &cachedPaste, nil
	}

	// стандарт
	if cached, ok := r.Cache.Get(slug); ok {
		if paste, valid := cached.(*model.Paste); valid {
			if paste.HasExpired() {
				r.Cache.Invalidate(slug)
				return nil, ErrPasteExpired
			}
			return paste, nil
		}
	}

	var paste model.Paste
	if err := r.DB.Where("slug = ?", slug).First(&paste).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPasteNotFound
		}
		return nil, err
	}
	if paste.HasExpired() {
		return nil, ErrPasteExpired
	}
	r.Cache.Set(slug, &paste, r.cacheTTL)
	return &paste, nil
}

func (r *PasteRepository) UpdatePaste(p *model.Paste) error {
	if err := p.Validate(); err != nil {
		return err
	}

	var exists model.Paste
	if err := r.DB.Where("slug = ?", p.Slug).First(&exists).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPasteNotFound
		}
		return err
	}

	if exists.HasExpired() {
		return ErrPasteExpired
	}

	p.UpdatedAt = time.Now()
	if err := r.DB.Save(p).Error; err != nil {
		return err
	}

	r.Cache.Set(p.Slug, p, r.cacheTTL)
	return nil
}

func (r *PasteRepository) IncrementViewCount(slug string) error {
	if _, err := r.GetPasteBySlug(slug); err != nil {
		return err
	}

	now := time.Now()

	if err := r.DB.Model(&model.Paste{}).Where("slug = ?", slug).
		Updates(map[string]interface{}{
			"view_count":  gorm.Expr("view_count + ?", 1),
			"last_viewed": now,
		}).Error; err != nil {
		return err
	}

	var paste model.Paste
	if err := r.DB.Where("slug = ?", slug).First(&paste).Error; err == nil {
		r.Cache.Set(slug, &paste, r.cacheTTL)
		return nil
	}

	if cached, ok := r.Cache.Get(slug); ok {
		if paste, valid := cached.(*model.Paste); valid {
			paste.ViewCount++
			if paste.LastViewed == nil {
				paste.LastViewed = &now
			} else {
				*paste.LastViewed = now
			}
			r.Cache.Set(slug, paste, r.cacheTTL)
		}
	}

	return nil
}

func (r *PasteRepository) GetTopPastes(limit int) ([]model.Paste, error) {
	var pastes []model.Paste
	if err := r.DB.Where("expires IS NULL OR expires > ?", time.Now()).
		Order("view_count DESC").
		Limit(limit).
		Find(&pastes).Error; err != nil {
		return nil, err
	}
	return pastes, nil
}

func (r *PasteRepository) GetRecentPastes(limit int) ([]model.Paste, error) {
	var pastes []model.Paste
	if err := r.DB.Where("expires IS NULL OR expires > ?", time.Now()).
		Order("created_at DESC").
		Limit(limit).
		Find(&pastes).Error; err != nil {
		return nil, err
	}
	return pastes, nil
}
