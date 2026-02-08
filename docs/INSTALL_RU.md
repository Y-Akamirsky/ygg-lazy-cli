# Установка YggLazy CLI

## Linux - Быстрая установка

```bash
curl -sL https://raw.githubusercontent.com/Y-Akamirsky/ygg-lazy-cli/main/install.sh | bash
```

**Примечание:** Установка займёт 1-5 минут, так как программа компилируется на вашем устройстве. Это обеспечивает совместимость со всеми дистрибутивами.

### Что делает установщик?

1. Проверяет наличие Go (устанавливает автоматически при необходимости)
2. Скачивает исходный код
3. Компилирует программу на вашем устройстве
4. Устанавливает в систему
5. Удаляет временные файлы

## macOS - Установка

### Вариант 1: Homebrew (Рекомендуется)

```bash
brew tap Y-Akamirsky/ygg-lazy-cli
brew install ygglazy
```

**Запуск:**
```bash
sudo ygglazy
```

Вот и всё! Homebrew всё настраивает автоматически, включая зависимости и PATH.

### Вариант 2: Скачать готовый бинарник

1. **Скачайте** подходящий бинарник для вашего Mac:
   - **Intel Mac**: Скачайте `ygglazy-darwin-amd64`
   - **Apple Silicon**: Скачайте `ygglazy-darwin-arm64`
   
   Из [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Снимите атрибут карантина macOS**:
   ```bash
   xattr -d com.apple.quarantine ~/Downloads/ygglazy-darwin-*
   ```
   
   > **Зачем?** macOS помечает скачанные файлы как помещённые в карантин. Эта команда снимает ограничение.

3. **Установите**:
   ```bash
   chmod +x ~/Downloads/ygglazy-darwin-*
   sudo mv ~/Downloads/ygglazy-darwin-* /usr/local/bin/ygglazy
   ```

4. **Запустите**:
   ```bash
   sudo ygglazy
   ```

### Вариант 3: Сборка из исходников

```bash
# Установите зависимости
brew install go git

# Клонируйте и соберите
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# Установите
sudo mv ygglazy /usr/local/bin/
```

### macOS - Решение проблем

**"Невозможно открыть, так как разработчик не может быть проверен"**
```bash
# Снимите атрибут карантина
xattr -d com.apple.quarantine /usr/local/bin/ygglazy
```

**Команда не найдена после установки**
```bash
# Проверьте, что /usr/local/bin в PATH
echo $PATH | grep /usr/local/bin

# Если нет, добавьте в конфиг оболочки:
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## BSD - Установка

### FreeBSD

1. **Скачайте**:
   ```bash
   fetch https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest/download/ygg-lazy-cli-freebsd-amd64
   ```

2. **Установите**:
   ```bash
   chmod +x ygglazy-freebsd-amd64
   sudo mv ygglazy-freebsd-amd64 /usr/local/bin/ygglazy
   ```

3. **Запустите**:
   ```bash
   sudo ygglazy
   ```

**Альтернатива: Сборка из исходников**
```bash
# Установите зависимости
sudo pkg install go git

# Клонируйте и соберите
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# Установите
sudo mv ygglazy /usr/local/bin/
```

### OpenBSD

1. **Скачайте**:
   ```bash
   ftp https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest/download/ygg-lazy-cli-openbsd-amd64
   ```

2. **Установите**:
   ```bash
   chmod +x ygglazy-openbsd-amd64
   doas mv ygglazy-openbsd-amd64 /usr/local/bin/ygglazy
   ```

3. **Запустите**:
   ```bash
   doas ygglazy
   ```

**Альтернатива: Сборка из исходников**
```bash
# Установите зависимости
doas pkg_add go git

# Клонируйте и соберите
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# Установите
doas mv ygglazy /usr/local/bin/
```

### NetBSD

1. **Скачайте**:
   ```bash
   ftp https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest/download/ygg-lazy-cli-netbsd-amd64
   ```

2. **Установите**:
   ```bash
   chmod +x ygglazy-netbsd-amd64
   su -c 'mv ygglazy-netbsd-amd64 /usr/local/bin/ygglazy'
   ```

3. **Запустите**:
   ```bash
   su -c ygglazy
   # или
   sudo ygglazy
   ```

**Альтернатива: Сборка из исходников**
```bash
# Установите зависимости (от root)
pkgin install go git

# Клонируйте и соберите (от пользователя)
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# Установите (от root)
su -c 'mv ygglazy /usr/local/bin/'
```

## Windows - Установка

1. **Скачайте** `ygglazy-windows-amd64.exe` или `ygglazy-windows-386.exe` из [Releases](https://github.com/Y-Akamirsky/ygg-lazy-cli/releases/latest)

2. **Запустите от имени администратора** (правый клик → "Запустить от имени администратора")

3. **Используйте** интерактивное меню

## Использование

### Linux/BSD
```bash
# Запустить (требуется root)
sudo ygglazy        # Linux, FreeBSD, NetBSD
doas ygglazy        # OpenBSD

# Показать версию (root не нужен)
ygglazy --version

# Справка
ygglazy --help
```

### macOS
```bash
# Запустить (требуется sudo)
sudo ygglazy

# Показать версию
ygglazy --version
```

### Windows
- Правый клик → "Запустить от имени администратора"
- Используйте PowerShell или CMD

## Удаление

### Linux (установлено через скрипт)
```bash
sudo ygglazy-uninstall
```

### macOS/BSD (ручная установка)
```bash
sudo rm /usr/local/bin/ygglazy        # FreeBSD, NetBSD, macOS
doas rm /usr/local/bin/ygglazy        # OpenBSD
```

### Windows
- Удалите файл .exe

## Решение проблем

### Linux: Нет git

```bash
# Debian/Ubuntu
sudo apt install git

# Fedora
sudo dnf install git

# Arch
sudo pacman -S git

# Alpine
sudo apk add git

# Void
sudo xbps-install -S git
```

### Linux: Программа не появляется в меню

```bash
sudo update-desktop-database /usr/share/applications/
```

### macOS: "разработчик не может быть проверен"

```bash
xattr -d com.apple.quarantine /usr/local/bin/ygg-lazy-cli
```

### BSD: Отсутствуют зависимости

```bash
# FreeBSD
sudo pkg install go git

# OpenBSD
doas pkg_add go git

# NetBSD
su -c 'pkgin install go git'
```

## Почему компиляция на устройстве? (Linux)

На старых ПК предкомпилированные бинарники могут не работать из-за несовместимости процессорных инструкций. Компиляция на вашем устройстве решает эту проблему.

## Ручная сборка из исходников (Все платформы)

Если автоматическая установка не работает:

```bash
# 1. Установите Go (если не установлен)
# Linux:
curl -sSL https://git.io/g-install | sh -s
source ~/.bashrc
g install latest

# macOS (Homebrew):
brew install go

# FreeBSD:
sudo pkg install go

# OpenBSD:
doas pkg_add go

# NetBSD:
su -c 'pkgin install go'

# 2. Скачайте и скомпилируйте
git clone https://github.com/Y-Akamirsky/ygg-lazy-cli.git
cd ygg-lazy-cli
go build -ldflags="-s -w" -trimpath -o ygglazy

# 3. Установите
sudo cp ygglazy /usr/local/bin/        # Linux, macOS, FreeBSD, NetBSD
doas cp ygglazy /usr/local/bin/        # OpenBSD
sudo chmod +x /usr/local/bin/ygglazy
```

## Особенности платформ

### macOS
- **Рекомендуется установка Yggdrasil через Homebrew**: `brew install yggdrasil-go`
- Расположение конфига: `/usr/local/etc/yggdrasil.conf` или `/opt/homebrew/etc/yggdrasil.conf`
- Управление сервисом через `launchctl`

### FreeBSD
- Установка Yggdrasil: `sudo pkg install yggdrasil` или через ports
- Расположение конфига: `/usr/local/etc/yggdrasil.conf`
- Управление сервисом: `sudo service yggdrasil start`

### OpenBSD
- Установка Yggdrasil: `doas pkg_add yggdrasil`
- Расположение конфига: `/etc/yggdrasil.conf`
- Управление сервисом: `doas rcctl start yggdrasil`

### NetBSD
- Установка Yggdrasil: `su -c 'pkgin install yggdrasil'`
- Расположение конфига: `/etc/yggdrasil.conf`
- Управление сервисом: `su -c '/etc/rc.d/yggdrasil start'`

## Поддержка

Если возникли проблемы, создайте [issue на GitHub](https://github.com/Y-Akamirsky/ygg-lazy-cli/issues) с указанием:
- ОС и версии (`uname -a` или `cat /etc/os-release`)
- Архитектуры (`uname -m`)
- Текста ошибки
- Использованного метода установки
