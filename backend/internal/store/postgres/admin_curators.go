package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) ListAdminCurators(actorUserID string, input model.ListAdminCuratorsInput) ([]*model.CuratorAccount, int, *commonstore.AppError) {
	ctx := context.Background()
	if _, _, repoErr := s.requireAdminCuratorActor(ctx, s.db, actorUserID); repoErr != nil {
		return nil, 0, repoErr
	}

	filterByActive := false
	filterActiveValue := false
	if input.IsActive != nil {
		filterByActive = true
		filterActiveValue = *input.IsActive
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.role, u.is_active,
			u.avatar_media_id, u.email_verified_at, u.last_login_at, u.token_version,
			u.created_at, u.updated_at, cp.full_name, cp.is_admin, cp.created_at
		FROM curator_profiles cp
		JOIN users u ON u.id = cp.user_id
		WHERE u.role = 'curator'
			AND ($1 = FALSE OR u.is_active = $2)
			AND (
				$3 = ''
				OR u.email ILIKE '%' || $3 || '%'
				OR u.display_name ILIKE '%' || $3 || '%'
				OR cp.full_name ILIKE '%' || $3 || '%'
			)
		ORDER BY cp.created_at DESC, u.id DESC
	`, filterByActive, filterActiveValue, strings.TrimSpace(input.Q))
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load curators", nil)
	}
	defer rows.Close()

	items := make([]*model.CuratorAccount, 0)
	for rows.Next() {
		item, scanErr := scanCuratorAccount(rows)
		if scanErr != nil {
			return nil, 0, appErr(500, "internal_error", "failed to load curators", nil)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load curators", nil)
	}

	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) CreateAdminCurator(actorUserID string, input model.CreateCuratorInput, passwordHash string) (*model.CuratorAccount, *commonstore.AppError) {
	if strings.TrimSpace(input.Reason) == "" {
		return nil, appErr(422, "validation_error", "reason is required", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create curator", nil)
	}
	defer tx.Rollback(ctx)

	if _, _, repoErr := s.requireAdminCuratorActor(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	now := time.Now().UTC()
	user, err := scanUser(tx.QueryRow(ctx, `
		INSERT INTO users (
			email, password_hash, display_name, role, is_active, token_version, created_at, updated_at
		) VALUES ($1, $2, $3, 'curator', TRUE, 0, $4, $4)
		RETURNING id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
	`, normalizeEmail(input.Email), passwordHash, strings.TrimSpace(input.DisplayName), now))
	if err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return nil, appErr(409, "conflict", "user with this email already exists", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create curator", nil)
	}

	profile, err := scanCuratorProfile(tx.QueryRow(ctx, `
		INSERT INTO curator_profiles (user_id, full_name, is_admin, created_at)
		VALUES ($1, $2, FALSE, $3)
		RETURNING user_id, full_name, is_admin, created_at
	`, user.ID, strings.TrimSpace(input.FullName), now))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create curator profile", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create curator", nil)
	}

	return &model.CuratorAccount{
		User:             user,
		FullName:         profile.FullName,
		IsAdmin:          profile.IsAdmin,
		ProfileCreatedAt: profile.CreatedAt,
	}, nil
}

func (s *Store) UpdateAdminCurator(actorUserID, targetUserID string, input model.UpdateCuratorAccountInput) (*model.CuratorAccount, *commonstore.AppError) {
	if strings.TrimSpace(input.Reason) == "" {
		return nil, appErr(422, "validation_error", "reason is required", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update curator", nil)
	}
	defer tx.Rollback(ctx)

	if _, _, repoErr := s.requireAdminCuratorActor(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	account, repoErr := s.loadCuratorAccount(ctx, tx, targetUserID, true)
	if repoErr != nil {
		return nil, repoErr
	}

	if account.IsAdmin && input.IsActive != nil && !*input.IsActive {
		activeAdmins, err := s.countActiveAdminCurators(ctx, tx)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to validate administrator invariant", nil)
		}
		if activeAdmins <= 1 {
			return nil, appErr(409, "conflict", "requested change would violate the single-administrator invariant", nil)
		}
	}

	userChanged := false
	if input.DisplayName != nil {
		account.User.DisplayName = strings.TrimSpace(*input.DisplayName)
		userChanged = true
	}
	if input.IsActive != nil {
		account.User.IsActive = *input.IsActive
		userChanged = true
	}
	if userChanged {
		account.User.UpdatedAt = time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET display_name = $1, is_active = $2, updated_at = $3
			WHERE id = $4
		`, account.User.DisplayName, account.User.IsActive, account.User.UpdatedAt, account.User.ID); err != nil {
			return nil, appErr(500, "internal_error", "failed to update curator user", nil)
		}
	}

	if input.FullName != nil {
		account.FullName = strings.TrimSpace(*input.FullName)
		if _, err := tx.Exec(ctx, `
			UPDATE curator_profiles
			SET full_name = $1
			WHERE user_id = $2
		`, account.FullName, account.User.ID); err != nil {
			return nil, appErr(500, "internal_error", "failed to update curator profile", nil)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update curator", nil)
	}

	return s.loadCuratorAccount(context.Background(), s.db, targetUserID, false)
}

func (s *Store) requireAdminCuratorActor(ctx context.Context, q queryable, actorUserID string) (*model.User, *curatorProfileRecord, *commonstore.AppError) {
	user, repoErr := s.requireCuratorActor(ctx, q, actorUserID)
	if repoErr != nil {
		return nil, nil, repoErr
	}
	profile, repoErr := s.requireCuratorProfile(ctx, q, actorUserID)
	if repoErr != nil {
		return nil, nil, repoErr
	}
	if !profile.IsAdmin {
		return nil, nil, appErr(403, "forbidden", "current user is not a platform administrator", nil)
	}
	return user, profile, nil
}

func (s *Store) loadCuratorAccount(ctx context.Context, q queryable, userID string, forUpdate bool) (*model.CuratorAccount, *commonstore.AppError) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.role, u.is_active,
			u.avatar_media_id, u.email_verified_at, u.last_login_at, u.token_version,
			u.created_at, u.updated_at, cp.full_name, cp.is_admin, cp.created_at
		FROM curator_profiles cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.user_id = $1
			AND u.role = 'curator'
	`
	if forUpdate {
		query += ` FOR UPDATE OF u, cp`
	}

	account, err := scanCuratorAccount(q.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "curator not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load curator", nil)
	}
	return account, nil
}

func (s *Store) countActiveAdminCurators(ctx context.Context, q queryable) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM curator_profiles cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.is_admin = TRUE
			AND u.role = 'curator'
			AND u.is_active = TRUE
	`).Scan(&count)
	return count, err
}

func scanCuratorAccount(src scanner) (*model.CuratorAccount, error) {
	var account model.CuratorAccount
	var user model.User
	var avatar sql.NullString
	var emailVerified sql.NullTime
	var lastLogin sql.NullTime
	if err := src.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.IsActive,
		&avatar,
		&emailVerified,
		&lastLogin,
		&user.TokenVersion,
		&user.CreatedAt,
		&user.UpdatedAt,
		&account.FullName,
		&account.IsAdmin,
		&account.ProfileCreatedAt,
	); err != nil {
		return nil, err
	}
	user.AvatarMediaID = nullStringPtr(avatar)
	user.EmailVerifiedAt = nullTimePtr(emailVerified)
	user.LastLoginAt = nullTimePtr(lastLogin)
	account.User = &user
	return &account, nil
}
