package main

import (
	"fmt"
	"os"
	"strings"
)

// Lang is a supported interface language.
type Lang string

const (
	LangEN Lang = "en"
	LangRU Lang = "ru"
)

var currentLang = LangEN

// ruCatalog maps English source strings to Russian translations. English is
// the fallback, so missing entries degrade gracefully.
var ruCatalog = map[string]string{
	// Help and version
	"YggLazy-cli version %s - Lazy way to configure Yggdrasil Network!": "YggLazy-cli версия %s — ленивый способ настроить сеть Yggdrasil!",
	"USAGE:":                               "ИСПОЛЬЗОВАНИЕ:",
	"OPTIONS:":                             "ПАРАМЕТРЫ:",
	"Show this help message":               "Показать эту справку",
	"Show version information":             "Показать информацию о версии",
	"Install Yggdrasil automatically":      "Автоматически установить Yggdrasil",
	"Use a custom yggdrasil.conf path":     "Использовать свой путь к yggdrasil.conf",
	"Set interface language: auto, en, ru": "Задать язык интерфейса: auto, en, ru",
	"EXAMPLES:":                            "ПРИМЕРЫ:",
	"Start interactive configurator":       "Запустить интерактивный конфигуратор",
	"Auto-install Yggdrasil":               "Автоустановка Yggdrasil",
	"Use a custom config file":             "Использовать свой файл конфигурации",
	"Show version (no sudo needed)":        "Показать версию (sudo не нужен)",
	"For more information, visit:":         "Подробнее:",
	"YggLazy version %s":                   "Версия YggLazy %s",
	"Built with Go %s":                     "Собрано с Go %s",
	"OS/Arch: %s/%s":                       "ОС/архитектура: %s/%s",

	// Main menu
	"Config loaded: %s":                "Загружен конфиг: %s",
	"Active peers in config: %d":       "Пиров в конфиге: %d",
	"Main Menu:":                       "Главное меню:",
	"Auto-select Peers (Best Latency)": "Автовыбор пиров (лучшая задержка)",
	"Manual Peer Selection":            "Ручной выбор пиров",
	"View Configured Peers":            "Показать настроенные пиры",
	"Check Active Peers Status":        "Проверить статус активных пиров",
	"Remove Dead Peers":                "Удалить мёртвые пиры",
	"Remove Peers":                     "Удалить пиры",
	"Add Custom Peer":                  "Добавить свой пир",
	"Node Status":                      "Статус узла",
	"Service Control":                  "Управление сервисом",
	"Exit":                             "Выход",
	"Bye!":                             "Пока!",
	"Back":                             "Назад",

	// Service menu
	"Service Control (Esc to back):": "Управление сервисом (Esc — назад):",
	"Start":                          "Запустить",
	"Stop":                           "Остановить",
	"Restart":                        "Перезапустить",
	"Enable Autostart":               "Включить автозапуск",
	"Disable Autostart":              "Отключить автозапуск",
	"Service operation failed: ":     "Ошибка управления сервисом: ",
	"Done.":                          "Готово.",
	"Restart Yggdrasil service now?": "Перезапустить сервис Yggdrasil сейчас?",
	"Restart failed: ":               "Ошибка перезапуска: ",
	"Service restarted.":             "Сервис перезапущен.",

	// Peers
	"=== Currently Configured Peers ===": "=== Настроенные пиры ===",
	"No peers found in config.":          "В конфиге нет пиров.",
	"No peers in config.":                "В конфиге нет пиров.",
	"Select to Remove:":                  "Выберите для удаления:",
	"Remove selected peers from config?": "Удалить выбранные пиры из конфига?",
	"Enter URIs (space separated):":      "Введите URI (через пробел):",
	"Failed to update config: ":          "Не удалось обновить конфиг: ",
	"Peers added and config formatted.":  "Пиры добавлены, конфиг отформатирован.",
	"Peers removed.":                     "Пиры удалены.",
	"Region:":                            "Регион:",
	"No peers.":                          "Нет пиров.",
	"Select Peers:":                      "Выберите пиров:",

	// Installation and config
	"Invalid config path: ": "Некорректный путь к конфигу: ",
	"Exiting.":              "Выход.",
	"Yggdrasil is not installed: no yggdrasil binary found in PATH.":                          "Yggdrasil не установлен: бинарник yggdrasil не найден в PATH.",
	"Automatic installation is experimental. Installing Yggdrasil separately is recommended.": "Автоустановка экспериментальная. Рекомендуется установить Yggdrasil отдельно.",
	"Install Yggdrasil now?":                                                     "Установить Yggdrasil сейчас?",
	"Yggdrasil installation did not complete. Exiting.":                          "Установка Yggdrasil не завершена. Выход.",
	"Config file not found: ":                                                    "Файл конфигурации не найден: ",
	"Check that the path is correct and that the file exists.":                   "Проверьте, что путь указан верно и файл существует.",
	"Generate yggdrasil.conf now?":                                               "Сгенерировать yggdrasil.conf сейчас?",
	"Generating config...":                                                       "Генерация конфига...",
	"Failed to generate config: ":                                                "Не удалось сгенерировать конфиг: ",
	"Generate it manually with:":                                                 "Сгенерируйте вручную командой:",
	"Config generated at %s":                                                     "Конфиг создан: %s",
	"Yggdrasil config not found in the default directory.":                       "Конфиг Yggdrasil не найден в стандартной директории.",
	"Expected path: %s":                                                          "Ожидаемый путь: %s",
	"Make sure that yggdrasil.conf is generated:":                                "Убедитесь, что yggdrasil.conf сгенерирован:",
	"Or specify a different config path with the --config|-c flag, for example:": "Или укажите другой путь к конфигу с помощью флага --config|-c, например:",
	"=== Yggdrasil Installer ===":                                                "=== Установщик Yggdrasil ===",
	"Installation failed: ":                                                      "Ошибка установки: ",
	"Failed to generate config automatically: ":                                  "Не удалось автоматически сгенерировать конфиг: ",
	"Check if Yggdrasil is in PATH, or generate it manually with:":               "Проверьте, что Yggdrasil есть в PATH, или сгенерируйте конфиг вручную:",
	"yggdrasil binary not found in PATH":                                         "бинарник yggdrasil не найден в PATH",
	"yggdrasil -genconf failed: %v":                                              "yggdrasil -genconf завершился с ошибкой: %v",
	"yggdrasil -genconf returned empty output":                                   "yggdrasil -genconf вернул пустой вывод",
	"Running: %s": "Выполняется: %s",

	// Platform installers
	"=== FreeBSD Yggdrasil Installation ===":                          "=== Установка Yggdrasil во FreeBSD ===",
	"=== NetBSD Yggdrasil Installation ===":                           "=== Установка Yggdrasil в NetBSD ===",
	"=== OpenBSD Yggdrasil Installation ===":                          "=== Установка Yggdrasil в OpenBSD ===",
	"=== macOS Yggdrasil Installation ===":                            "=== Установка Yggdrasil в macOS ===",
	"Administrator privileges required. Relaunching...":               "Требуются права администратора. Перезапуск...",
	"Detected Alpine Linux. Using apk...":                             "Обнаружен Alpine Linux. Используется apk...",
	"Detected Arch-based system. Attempting to install via pacman...": "Обнаружена система на базе Arch. Попытка установки через pacman...",
	"Detected Distro: %s (Like: %s)":                                  "Обнаружен дистрибутив: %s (Like: %s)",
	"Detected Fedora-based system. Using DNF and COPR...":             "Обнаружена система на базе Fedora. Используется DNF и COPR...",
	"Detected Void Linux. Using xbps-install...":                      "Обнаружен Void Linux. Используется xbps-install...",
	"Download .deb":                                                 "Скачать .deb",
	"Download .pkg":                                                 "Скачать .pkg",
	"Downloading %s (Arch: %s)...":                                  "Загрузка %s (архитектура: %s)...",
	"Downloading %s...":                                             "Загрузка %s...",
	"Downloading .pkg installer...":                                 "Загрузка установщика .pkg...",
	"Enable service: rcctl enable yggdrasil":                        "Включить сервис: rcctl enable yggdrasil",
	"Enable service: sysrc yggdrasil_enable=YES":                    "Включить сервис: sysrc yggdrasil_enable=YES",
	"Executing: %s":                                                 "Выполняется: %s",
	"Failed to elevate: ":                                           "Не удалось повысить права: ",
	"Failed to install %s. Trying next...":                          "Не удалось установить %s. Пробуем следующий...",
	"Fetching latest release...":                                    "Получение последнего релиза...",
	"Homebrew (recommended)":                                        "Homebrew (рекомендуется)",
	"Homebrew detected.":                                            "Обнаружен Homebrew.",
	"Installation complete.":                                        "Установка завершена.",
	"Installed successfully via APK.":                               "Успешно установлено через APK.",
	"Installed successfully via XBPS.":                              "Успешно установлено через XBPS.",
	"Installed successfully.":                                       "Успешно установлено.",
	"Installing package...":                                         "Установка пакета...",
	"Installing via Homebrew...":                                    "Установка через Homebrew...",
	"Installing via pkg...":                                         "Установка через pkg...",
	"Installing via pkg_add...":                                     "Установка через pkg_add...",
	"Installing via pkgin...":                                       "Установка через pkgin...",
	"Installing via pkgsrc...":                                      "Установка через pkgsrc...",
	"Installing via ports...":                                       "Установка через ports...",
	"Note: To start service, use: brew services start yggdrasil-go": "Примечание: для запуска сервиса используйте: brew services start yggdrasil-go",
	"Please add 'yggdrasil=YES' to /etc/rc.conf manually":           "Пожалуйста, добавьте 'yggdrasil=YES' в /etc/rc.conf вручную",
	"Please remove 'yggdrasil=YES' from /etc/rc.conf manually":      "Пожалуйста, удалите 'yggdrasil=YES' из /etc/rc.conf вручную",
	"Ports tree not found. Please install it first:":                "Дерево ports не найдено. Сначала установите его:",
	"Running MSI installer (Admin rights required)...":              "Запуск MSI-установщика (требуются права администратора)...",
	"Service will auto-start after load.":                           "Сервис будет автоматически запускаться после загрузки.",
	"Start service: rcctl start yggdrasil":                          "Запустить сервис: rcctl start yggdrasil",
	"Start service: service yggdrasil start":                        "Запустить сервис: service yggdrasil start",
	"Trying package: %s":                                            "Пробуем пакет: %s",
	"Use APT Repository":                                            "Использовать репозиторий APT",
	"apk failed":                                                    "apk завершился с ошибкой",
	"brew install failed":                                           "brew install завершился с ошибкой",
	"could not install yggdrasil via pacman. Try installing manually from AUR": "не удалось установить yggdrasil через pacman. Попробуйте установить вручную из AUR",
	"installation cancelled":                          "установка отменена",
	"pkg (binary package)":                            "pkg (бинарный пакет)",
	"pkg install failed":                              "pkg install завершился с ошибкой",
	"pkg_add failed":                                  "pkg_add завершился с ошибкой",
	"pkgin (binary package)":                          "pkgin (бинарный пакет)",
	"pkgin install failed":                            "pkgin install завершился с ошибкой",
	"pkgsrc (compile from source)":                    "pkgsrc (сборка из исходников)",
	"pkgsrc install failed":                           "pkgsrc install завершился с ошибкой",
	"pkgsrc tree not available":                       "дерево pkgsrc недоступно",
	"pkgsrc tree not found. Please install it first.": "Дерево pkgsrc не найдено. Сначала установите его.",
	"ports (compile from source)":                     "ports (сборка из исходников)",
	"ports install failed":                            "ports install завершился с ошибкой",
	"ports tree not available":                        "дерево ports недоступно",
	"unknown action":                                  "неизвестное действие",
	"unsupported distribution: %s":                    "неподдерживаемый дистрибутив: %s",
	"xbps-install failed":                             "xbps-install завершился с ошибкой",

	// Peer discovery and testing
	"Scanning repository...":                   "Сканирование репозитория...",
	"Error: ":                                  "Ошибка: ",
	"Select region to test peers from:":        "Выберите регион для тестирования пиров:",
	"All regions":                              "Все регионы",
	"Cancel":                                   "Отмена",
	"Fetching from %d regions...":              "Загрузка из %d регионов...",
	"Fetching peers from %s region...":         "Загрузка пиров из региона %s...",
	"Total peers found: %d. Testing all...":    "Найдено пиров: %d. Тестируем все...",
	"Testing %d peers with 5 attempts each...": "Тестирование %d пиров по 5 попыток...",
	"Note: Final peer verification happens after adding them to config.":             "Примечание: окончательная проверка пиров произойдёт после добавления в конфиг.",
	"Use 'Check Active Peers Status' to verify and 'Remove Dead Peers' to clean up.": "Используйте «Проверить статус активных пиров» для проверки и «Удалить мёртвые пиры» для очистки.",
	"[%d/%d] ✓ Found %d peers (last: %s, jitter: %s)":                                "[%d/%d] ✓ Найдено пиров: %d (последний: %s, джиттер: %s)",
	"[%d/%d] ✗ Testing... (%d peers found)":                                          "[%d/%d] ✗ Проверка... (найдено пиров: %d)",
	"=== Testing Summary ===":                                                        "=== Итоги тестирования ===",
	"Total peers tested: %d":                                                         "Протестировано пиров: %d",
	"Peers found: %s":                                                                "Найдено пиров: %s",
	"Best latency: %s":                                                               "Лучшая задержка: %s",
	"Best stability: %.2f%%":                                                         "Лучшая стабильность: %.2f%%",
	"No reachable peers found with latency < 5s.":                                    "Не найдено доступных пиров с задержкой < 5 с.",
	"This might mean:":                                                               "Возможные причины:",
	"  - Network connectivity issues":                                                "  - Проблемы с сетью",
	"  - Firewall blocking connections":                                              "  - Файрвол блокирует подключения",
	"  - All peers are currently down":                                               "  - Все пиры сейчас недоступны",
	"Top Peers by Latency & Stability:":                                              "Лучшие пиры по задержке и стабильности:",
	"%d. %s\n   Latency: %s (min: %s, max: %s, jitter: %s) - %s (%.0f%%)\n":          "%d. %s\n   Задержка: %s (мин: %s, макс: %s, джиттер: %s) — %s (%.0f%%)\n",
	"excellent":                  "отличная",
	"good":                       "хорошая",
	"fair":                       "средняя",
	"unstable":                   "нестабильная",
	"(+%d more peers available)": "(ещё доступно пиров: %d)",
	"How many of the best peers would you like to add?": "Сколько лучших пиров добавить?",
	"3 peers (recommended)":                             "3 пира (рекомендуется)",
	"5 peers":                                           "5 пиров",
	"7 peers":                                           "7 пиров",
	"10 peers":                                          "10 пиров",
	"Custom number":                                     "Своё число",
	"Enter number of peers to add (1-%d):":              "Введите число пиров для добавления (1-%d):",
	"Invalid number. Canceling.":                        "Неверное число. Отмена.",
	"Peers to add:":                                     "Пиры для добавления:",
	"%d. %s (%s avg, jitter: %s)":                       "%d. %s (среднее: %s, джиттер: %s)",
	"Confirm adding these peers?":                       "Добавить этих пиров?",

	// Status and dead peer cleanup
	"=== Active Peers Status ===\n":                                                "=== Статус активных пиров ===\n",
	"Make sure Yggdrasil service is running.":                                      "Убедитесь, что сервис Yggdrasil запущен.",
	"\nTip: Use 'Remove Dead Peers' to clean up peers with 'Down' status.":         "\nСовет: используйте «Удалить мёртвые пиры», чтобы убрать пиры со статусом «Down».",
	"=== Remove Dead Peers ===\n":                                                  "=== Удаление мёртвых пиров ===\n",
	"Fetching peer status from Yggdrasil...":                                       "Получение статуса пиров из Yggdrasil...",
	"Error getting peer status: ":                                                  "Ошибка получения статуса пиров: ",
	"Error parsing peers data: ":                                                   "Ошибка разбора данных о пирах: ",
	"No peer data found in response.":                                              "В ответе нет данных о пирах.",
	"Make sure Yggdrasil service is running and has been started with some peers.": "Убедитесь, что сервис Yggdrasil запущен и в конфиге есть пиры.",
	"No active peer connections found.":                                            "Активные подключения к пирам не найдены.",
	"Yggdrasil might still be starting up or no peers are configured.":             "Yggdrasil мог ещё не запуститься, или пиры не настроены.",
	"\nFound %d peer connection(s) in Yggdrasil. Analyzing...\n\n":                 "\nНайдено подключений к пирам: %d. Анализ...\n\n",
	"Up":   "Работают",
	"Down": "Не работают",
	"✓ All peers are up! No dead peers to remove.": "✓ Все пиры работают! Мёртвых пиров нет.",
	"Dead peers (status: Down):":                   "Мёртвые пиры (статус: Down):",
	"   Error: %s\n":                               "   Ошибка: %s\n",
	"Note: Some dead peers are not in config (possibly added temporarily):": "Примечание: некоторые мёртвые пиры отсутствуют в конфиге (возможно, добавлены временно):",
	"No dead peers found in config to remove.":                              "В конфиге нет мёртвых пиров для удаления.",
	"(All dead peers were added temporarily, not via config)":               "(Все мёртвые пиры были добавлены временно, не через конфиг)",
	"Found %s dead peers in config:":                                        "Найдено мёртвых пиров в конфиге: %s",
	"Remove these dead peers from config?":                                  "Удалить этих мёртвых пиров из конфига?",
	"\n✓ Dead peers removed from config.":                                   "\n✓ Мёртвые пиры удалены из конфига.",

	// General
	"Press any key or click to continue...": "Нажмите любую клавишу или кликните мышью...",
	"Configurator v":                        "Конфигуратор v",
	"Select language:":                      "Выберите язык:",
	"Language changed.":                     "Язык изменён.",
	"Use arrows or mouse to move, enter to confirm, type to filter":                "Стрелки/мышь — выбор, Enter — подтвердить, ввод — фильтр",
	"Use arrows or mouse to move, space to mark, enter to confirm, type to filter": "Стрелки/мышь — выбор, Пробел — отметить, Enter — подтвердить, ввод — фильтр",

	// Confirmation
	"(Y/n)": "(Д/н)",
	"(y/N)": "(д/Н)",
	"Yes":   "Да",
	"No":    "Нет",
	"%q is not a valid answer, please try again.": "%q — неверный ответ, попробуйте снова.",
}

// T returns the translated string for the current language. The English source
// string is used as the catalog key, so missing translations fall back to it.
func T(key string, args ...any) string {
	text := key
	if currentLang == LangRU {
		if translated, ok := ruCatalog[key]; ok {
			text = translated
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

// detectLanguage guesses the interface language from the OS locale variables.
func detectLanguage() Lang {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		value := os.Getenv(name)
		if value == "" {
			continue
		}
		for _, part := range strings.Split(value, ":") {
			part = strings.ToLower(strings.TrimSpace(part))
			if idx := strings.IndexAny(part, "._@"); idx != -1 {
				part = part[:idx]
			}
			if part == "ru" {
				return LangRU
			}
		}
	}
	return LangEN
}

// setLanguage applies "auto", "en" or "ru" as the interface language.
func setLanguage(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		currentLang = detectLanguage()
	case "en", "english":
		currentLang = LangEN
	case "ru", "russian", "русский":
		currentLang = LangRU
	default:
		return fmt.Errorf("unsupported language %q (use auto, en or ru)", value)
	}
	return nil
}

// parseConfirmAnswer accepts English and Russian yes/no answers.
func parseConfirmAnswer(answer string, defaultValue bool) (value bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "":
		return defaultValue, true
	case "y", "yes", "д", "да":
		return true, true
	case "n", "no", "н", "нет":
		return false, true
	default:
		return false, false
	}
}

// askConfirm asks a yes/no question with a localized prompt.
func askConfirm(message string, defaultValue bool) bool {
	answer, err := promptConfirm(T(message), defaultValue)
	if err != nil {
		if err == errPromptInterrupt {
			fmt.Println(T("Bye!"))
			os.Exit(0)
		}
		return false
	}
	return answer
}
