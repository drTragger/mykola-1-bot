# 🍓 mykola-1-bot

**mykola-1-bot** — це Telegram-бот для керування домашнім сервером на Raspberry Pi.

Він дає швидкий доступ до метрик системи, стану сервісів і (в перспективі) керування медіа — прямо з Telegram.

---

## ⚡ Що він вміє

### 📊 Системні метрики
- CPU usage + кількість ядер
- CPU frequency (важливо для Raspberry Pi)
- RAM / ZRAM / SWAP
- Disk usage + вільне місце
- Температура CPU
- Load average
- Uptime / Boot time
- Кількість процесів і користувачів

---

### 🍓 Raspberry Pi health
- Throttling (перегрів / undervoltage)
- Обмеження частоти CPU
- Історія проблем (було / є зараз)

---

### 🌐 Мережа
- RX / TX (загальний трафік)
- Швидкість (в реальному часі)
- Ping
- Public IP

---

### 🧩 Сервіси (systemd)
Моніторинг стану:
- Jellyfin
- qBittorrent
- Sonarr
- Radarr
- Prowlarr
- Fail2Ban

---

### 🤖 Telegram команди

- `/metrics` — повний звіт
- `/simple` — спрощений режим
- `/help` — список команд

---

## 🏗️ Архітектура

Проєкт написаний на Go з мінімальною кількістю залежностей.