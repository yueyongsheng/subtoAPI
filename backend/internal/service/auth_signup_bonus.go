package service

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

const signupBonusCodePrefix = "signup-"

func signupBonusCode(userID int64) string {
	return fmt.Sprintf("%s%d", signupBonusCodePrefix, userID)
}

func (s *AuthService) createUserWithSignupBonus(ctx context.Context, user *User) error {
	if user == nil {
		return nil
	}
	if user.Balance <= 0 || s.redeemRepo == nil {
		return s.userRepo.Create(ctx, user)
	}

	// Reuse a caller-owned transaction when registration also consumes an
	// invitation code. Otherwise, make the user and history record atomic here.
	if s.entClient == nil || dbent.TxFromContext(ctx) != nil {
		return s.createUserAndSignupBonusRecord(ctx, user)
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.createUserAndSignupBonusRecord(txCtx, user); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *AuthService) createUserAndSignupBonusRecord(ctx context.Context, user *User) error {
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	return s.createSignupBonusRecord(ctx, user)
}

func (s *AuthService) createSignupBonusRecord(ctx context.Context, user *User) error {
	if user.Balance <= 0 || s.redeemRepo == nil {
		return nil
	}

	now := time.Now().UTC()
	return s.redeemRepo.Create(ctx, &RedeemCode{
		Code:   signupBonusCode(user.ID),
		Type:   RedeemTypeSignupBonus,
		Value:  user.Balance,
		Status: StatusUsed,
		UsedBy: &user.ID,
		UsedAt: &now,
	})
}

func (s *AuthService) deleteSignupBonusRecord(ctx context.Context, userID int64) error {
	if s == nil || s.redeemRepo == nil || userID <= 0 {
		return nil
	}

	record, err := s.redeemRepo.GetByCode(ctx, signupBonusCode(userID))
	if err != nil {
		if err == ErrRedeemCodeNotFound {
			return nil
		}
		return err
	}
	if record.Type != RedeemTypeSignupBonus || record.UsedBy == nil || *record.UsedBy != userID {
		return nil
	}
	return s.redeemRepo.Delete(ctx, record.ID)
}
