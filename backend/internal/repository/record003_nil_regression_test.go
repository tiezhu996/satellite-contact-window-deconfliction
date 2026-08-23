package repository_test

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/dto"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
	"satellite-contact-window-deconfliction/backend/internal/service"
)

func openRecord003DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:record003?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	return db
}

func TestParseTokenInvalidNoPanicP301(t *testing.T) {
	db := openRecord003DB(t)
	auth := service.NewAuthService(repository.NewSystemRepository(db), "record003-secret-at-least-32-bytes", time.Hour)
	_, err := auth.ParseToken("definitely-not-a-jwt-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestLoginInactiveUserRejectedP302(t *testing.T) {
	db := openRecord003DB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("password-527"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "retired-user", PasswordHash: string(hash), Role: "scheduler", Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("active", false).Error; err != nil {
		t.Fatal(err)
	}
	auth := service.NewAuthService(repository.NewSystemRepository(db), "record003-secret-at-least-32-bytes", time.Hour)
	if _, err := auth.Login(dto.LoginRequest{Username: "retired-user", Password: "password-527"}); err == nil {
		t.Fatal("expected inactive user login to be rejected")
	}
}

func TestFindUserUnknownReturnsErrorP303(t *testing.T) {
	db := openRecord003DB(t)
	repo := repository.NewSystemRepository(db)
	if _, err := repo.FindUser("ghost-user"); err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestFindActiveUserByIDZeroRejectedP304(t *testing.T) {
	db := openRecord003DB(t)
	repo := repository.NewSystemRepository(db)
	if _, err := repo.FindActiveUserByID(0); err == nil {
		t.Fatal("expected error for zero user id")
	}
}
