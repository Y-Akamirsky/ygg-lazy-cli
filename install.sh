#!/bin/bash

# Проверка на root
if [ "$EUID" -ne 0 ]; then
  echo "Пожалуйста, запустите установщик с sudo (нужно для копирования в /usr/local/bin)"
  exit
fi

REPO="Y-Akamirsky/ygg-lazy-cli"
# Автоопределение архитектуры для скачивания нужного бинарника
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
  BINARY_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli-amd64"
elif [[ "$ARCH" == "aarch64" ]]; then
  BINARY_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli-arm64"
else
  echo "Архитектура $ARCH не поддерживается... пока что."
  exit 1
fi

ICON_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygglazycli.svg"
DESKTOP_URL="https://raw.githubusercontent.com/$REPO/main/linux-install/ygg-lazy-cli.desktop"

echo "=== Установка YggLazy-cli ==="

# 1. Скачиваем бинарник
echo "Скачивание бинарника..."
curl -L -o /usr/local/bin/ygg-lazy-cli "$BINARY_URL"
chmod +x /usr/local/bin/ygg-lazy-cli

# 2. Скачиваем иконку
echo "Установка иконки..."
mkdir -p /usr/local/share/icons
curl -L -o /usr/local/share/icons/ygglazycli.svg "$ICON_URL"

# 3. Скачиваем .desktop файл
echo "Создание ярлыка меню..."
curl -L -o /usr/share/applications/ygg-lazy-cli.desktop "$DESKTOP_URL"

# Обновляем кэш десктоп файлов (чтобы иконка появилась сразу)
update-desktop-database /usr/share/applications/ 2>/dev/null

echo "Готово! Теперь ищите YggLazy-cli в меню приложений."
