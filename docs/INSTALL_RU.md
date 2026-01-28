# Установка YggLazy CLI

## Быстрая установка

```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

**Примечание:** Установка займёт 1-5 минут, так как программа компилируется на вашем устройстве. Это обеспечивает совместимость со всеми дистрибутивами.

## Что делает установщик?

1. Проверяет наличие Go (устанавливает автоматически при необходимости)
2. Скачивает исходный код
3. Компилирует программу на вашем устройстве
4. Устанавливает в систему
5. Удаляет временные файлы

## Использование

```bash
# Запустить (требуется sudo)
sudo ygg-lazy-cli

# Показать версию
ygg-lazy-cli --version

# Справка
ygg-lazy-cli --help
```

## Удаление

```bash
sudo ygg-lazy-cli-uninstall
```

## Решение проблем

### Go не устанавливается автоматически

```bash
# Установите Go вручную через утилиту 'g' если ваш дистрибутив не поставляет его или версия в репозитории устарела
curl -sSL https://git.io/g-install | sh -s
source ~/.bashrc
g install latest
```
```bash
# Затем повторите установку
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | sudo bash
```

### Нет git

```bash
# Debian/Ubuntu
sudo apt install git

# Fedora
sudo dnf install git

# Arch
sudo pacman -S git
```

### Программа не появляется в меню

```bash
sudo update-desktop-database /usr/share/applications/
```

## Почему компиляция на устройстве?

На старых ПК предкомпилированные бинарники могут не работать из-за несовместимости процессорных инструкций. Компиляция на вашем устройстве решает эту проблему.

## Ручная установка

Если автоматический скрипт не работает:

```bash
# 1. Установите Go
curl -sSL https://git.io/g-install | sh -s
source ~/.bashrc
g install latest

# 2. Скачайте и скомпилируйте
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o ygg-lazy-cli .

# 3. Установите
sudo cp ygg-lazy-cli /usr/local/bin/
sudo chmod +x /usr/local/bin/ygg-lazy-cli
```

## Поддержка

Если возникли проблемы, создайте [issue на GitHub](https://github.com/Y-Akamirsky/ygg-lazy-cli/issues) с указанием:
- Версии дистрибутива (`cat /etc/os-release`)
- Архитектуры (`uname -m`)
- Текста ошибки
