package main

import "testing"

func TestTranslationSwitchesLanguage(t *testing.T) {
	oldLang := currentLang
	defer func() { currentLang = oldLang }()

	if err := setLanguage("ru"); err != nil {
		t.Fatalf("setLanguage(ru): %v", err)
	}
	if got := T("Exit"); got != "Выход" {
		t.Fatalf("T(Exit) in Russian = %q, want %q", got, "Выход")
	}
	if got := T("Config loaded: %s", "/tmp/yggdrasil.conf"); got != "Загружен конфиг: /tmp/yggdrasil.conf" {
		t.Fatalf("formatted translation = %q", got)
	}

	if err := setLanguage("en"); err != nil {
		t.Fatalf("setLanguage(en): %v", err)
	}
	if got := T("Exit"); got != "Exit" {
		t.Fatalf("T(Exit) in English = %q, want %q", got, "Exit")
	}
}

func TestDetectLanguageFromLocale(t *testing.T) {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		t.Setenv(name, "")
	}

	t.Setenv("LANG", "ru_RU.UTF-8")
	if got := detectLanguage(); got != LangRU {
		t.Fatalf("detectLanguage() = %q, want %q", got, LangRU)
	}

	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LANGUAGE", "ru:en")
	if got := detectLanguage(); got != LangRU {
		t.Fatalf("detectLanguage() with LANGUAGE = %q, want %q", got, LangRU)
	}
}

func TestSetLanguageRejectsUnknown(t *testing.T) {
	if err := setLanguage("de"); err == nil {
		t.Fatal("expected an error for an unsupported language")
	}
}

func TestParseConfirmAnswerBilingual(t *testing.T) {
	tests := []struct {
		answer string
		def    bool
		want   bool
		wantOK bool
	}{
		{"y", false, true, true},
		{"YES", false, true, true},
		{"д", false, true, true},
		{"Да", false, true, true},
		{"n", true, false, true},
		{"NO", true, false, true},
		{"н", true, false, true},
		{"нет", true, false, true},
		{"", true, true, true},
		{"", false, false, true},
		{"maybe", true, false, false},
	}

	for _, tt := range tests {
		got, ok := parseConfirmAnswer(tt.answer, tt.def)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("parseConfirmAnswer(%q, %v) = (%v, %v), want (%v, %v)",
				tt.answer, tt.def, got, ok, tt.want, tt.wantOK)
		}
	}
}
