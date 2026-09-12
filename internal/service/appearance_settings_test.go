package service

import (
	"strings"
	"testing"

	"github.com/xuzhougeng/citebox/internal/model"
)

func appearanceStrPtr(s string) *string { return &s }

func TestGetAppearanceSettingsDefaultsToEmpty(t *testing.T) {
	svc, _, _ := newTestService(t)

	settings, err := svc.GetAppearanceSettings()
	if err != nil {
		t.Fatalf("GetAppearanceSettings() error = %v", err)
	}
	if settings.Theme != "" || settings.Language != "" {
		t.Fatalf("GetAppearanceSettings() = %+v, want empty defaults", settings)
	}
}

func TestUpdateAppearanceSettingsPersistsAndNormalizes(t *testing.T) {
	svc, repo, _ := newTestService(t)

	updated, err := svc.UpdateAppearanceSettings(model.AppearanceSettingsUpdate{
		Theme:    appearanceStrPtr(model.AppearanceThemeLight),
		Language: appearanceStrPtr(model.AppearanceLangEn),
	})
	if err != nil {
		t.Fatalf("UpdateAppearanceSettings() error = %v", err)
	}
	if updated.Theme != model.AppearanceThemeLight || updated.Language != model.AppearanceLangEn {
		t.Fatalf("UpdateAppearanceSettings() = %+v, want light/en", updated)
	}

	reloaded, err := svc.GetAppearanceSettings()
	if err != nil {
		t.Fatalf("GetAppearanceSettings() reload error = %v", err)
	}
	if reloaded.Theme != model.AppearanceThemeLight || reloaded.Language != model.AppearanceLangEn {
		t.Fatalf("GetAppearanceSettings() reload = %+v, want persisted light/en", reloaded)
	}

	raw, err := repo.GetAppSetting(appearanceSettingsKey)
	if err != nil {
		t.Fatalf("GetAppSetting(%q) error = %v", appearanceSettingsKey, err)
	}
	if !strings.Contains(raw, `"theme":"light"`) || !strings.Contains(raw, `"language":"en"`) {
		t.Fatalf("saved appearance settings = %q, want light/en persisted", raw)
	}

	invalid, err := svc.UpdateAppearanceSettings(model.AppearanceSettingsUpdate{
		Theme: appearanceStrPtr("neon"),
	})
	if err != nil {
		t.Fatalf("UpdateAppearanceSettings(invalid) error = %v", err)
	}
	if invalid.Theme != "" {
		t.Fatalf("UpdateAppearanceSettings(invalid) theme = %q, want unsupported value dropped", invalid.Theme)
	}
}

func TestUpdateAppearanceSettingsPartialKeepsOtherField(t *testing.T) {
	svc, _, _ := newTestService(t)

	if _, err := svc.UpdateAppearanceSettings(model.AppearanceSettingsUpdate{Theme: appearanceStrPtr(model.AppearanceThemeDark)}); err != nil {
		t.Fatalf("UpdateAppearanceSettings(theme) error = %v", err)
	}

	updated, err := svc.UpdateAppearanceSettings(model.AppearanceSettingsUpdate{Language: appearanceStrPtr(model.AppearanceLangEn)})
	if err != nil {
		t.Fatalf("UpdateAppearanceSettings(language) error = %v", err)
	}
	if updated.Theme != model.AppearanceThemeDark || updated.Language != model.AppearanceLangEn {
		t.Fatalf("UpdateAppearanceSettings(language) = %+v, want theme kept and language set", updated)
	}
}

func TestUpdateAppearanceSettingsEmptyResets(t *testing.T) {
	svc, repo, _ := newTestService(t)

	if _, err := svc.UpdateAppearanceSettings(model.AppearanceSettingsUpdate{
		Theme:    appearanceStrPtr(model.AppearanceThemeDark),
		Language: appearanceStrPtr(model.AppearanceLangZhCN),
	}); err != nil {
		t.Fatalf("UpdateAppearanceSettings() error = %v", err)
	}

	if _, err := svc.UpdateAppearanceSettings(model.AppearanceSettingsUpdate{
		Theme:    appearanceStrPtr(""),
		Language: appearanceStrPtr(""),
	}); err != nil {
		t.Fatalf("UpdateAppearanceSettings(clear) error = %v", err)
	}

	raw, err := repo.GetAppSetting(appearanceSettingsKey)
	if err != nil {
		t.Fatalf("GetAppSetting(%q) error = %v", appearanceSettingsKey, err)
	}
	if strings.TrimSpace(raw) != "" {
		t.Fatalf("GetAppSetting(%q) = %q, want empty after reset", appearanceSettingsKey, raw)
	}
}
