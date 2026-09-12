package service

import (
	"encoding/json"
	"strings"

	"github.com/xuzhougeng/citebox/internal/apperr"
	"github.com/xuzhougeng/citebox/internal/model"
)

const appearanceSettingsKey = "appearance_settings"

func (s *LibraryService) GetAppearanceSettings() (*model.AppearanceSettings, error) {
	raw, err := s.repo.GetAppSetting(appearanceSettingsKey)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "读取外观配置失败", err)
	}

	settings := model.AppearanceSettings{}
	if strings.TrimSpace(raw) == "" {
		return &settings, nil
	}

	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "解析外观配置失败", err)
	}

	normalized := normalizeAppearanceSettings(settings)
	return &normalized, nil
}

// UpdateAppearanceSettings merges the given update into the stored
// appearance settings: nil fields are left unchanged, and setting a field to
// an empty/unsupported value clears it.
func (s *LibraryService) UpdateAppearanceSettings(input model.AppearanceSettingsUpdate) (*model.AppearanceSettings, error) {
	current := model.AppearanceSettings{}

	raw, err := s.repo.GetAppSetting(appearanceSettingsKey)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "读取外观配置失败", err)
	}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &current); err != nil {
			return nil, apperr.Wrap(apperr.CodeInternal, "解析外观配置失败", err)
		}
		current = normalizeAppearanceSettings(current)
	}

	if input.Theme != nil {
		current.Theme = model.NormalizeAppearanceTheme(strings.TrimSpace(*input.Theme))
	}
	if input.Language != nil {
		current.Language = model.NormalizeAppearanceLanguage(strings.TrimSpace(*input.Language))
	}

	if current.Theme == "" && current.Language == "" {
		if err := s.repo.DeleteAppSetting(appearanceSettingsKey); err != nil {
			return nil, apperr.Wrap(apperr.CodeInternal, "保存外观配置失败", err)
		}
		return &current, nil
	}

	payload, err := json.Marshal(current)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "序列化外观配置失败", err)
	}
	if err := s.repo.UpsertAppSetting(appearanceSettingsKey, string(payload)); err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "保存外观配置失败", err)
	}

	return &current, nil
}

func normalizeAppearanceSettings(input model.AppearanceSettings) model.AppearanceSettings {
	return model.AppearanceSettings{
		Theme:    model.NormalizeAppearanceTheme(strings.TrimSpace(input.Theme)),
		Language: model.NormalizeAppearanceLanguage(strings.TrimSpace(input.Language)),
	}
}
