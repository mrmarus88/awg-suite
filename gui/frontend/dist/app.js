// Frontend logic for the AmneziaWG Manager. Backend methods are exposed by Wails
// at window.go.main.App.*; runtime events arrive via window.runtime.EventsOn.

const $ = (id) => document.getElementById(id);
const backend = () => window.go.main.App;

const APP_URLS = {
  ios: "https://apps.apple.com/app/amneziawg/id6478942365",
  android: "https://play.google.com/store/apps/details?id=org.amnezia.awg",
  macos: "https://apps.apple.com/app/amneziawg/id6478942365",
  windows: "https://github.com/amnezia-vpn/amneziawg-windows-client/releases",
};

const RENT_URLS = {
  vdsina: "https://www.vdsina.com/?partner=7yhz21p6dkml",
  hshp: "https://hshp.host/?from=144227",
};

// --- i18n ------------------------------------------------------------------

const I18N = {
  ru: {
    switch_server: "Сменить сервер",
    connect_title: "Подключение к серверу",
    connect_sub: "Введите данные вашего Linux-сервера. Приложение подключится по SSH и всё сделает само.",
    howto_summary: "Как это работает? (3 шага)",
    howto_1: "<b>Сервер.</b> Нужен любой VPS на Ubuntu/Debian (например, у хостинг-провайдера) — его IP, логин (обычно <span class=\"mono\">root</span>) и пароль.",
    howto_2: "<b>Установка.</b> Подключитесь здесь — приложение само поставит AmneziaWG и создаст первого клиента.",
    howto_3: "<b>Подключение.</b> Установите приложение AmneziaWG на телефон/ПК и отсканируйте QR (или импортируйте файл .conf).",
    rent_lead: "Нет своего сервера? Можно арендовать VPS:",
    rent_note: "(партнёрские ссылки — поддерживают проект)",
    saved_servers: "Сохранённые серверы",
    lbl_host: "IP-адрес сервера",
    lbl_user: "Пользователь",
    lbl_label: "Название (необязательно)",
    ph_label: "напр. VDSina, Германия",
    auth_password: "Пароль",
    auth_key: "SSH-ключ",
    lbl_password: "Пароль",
    lbl_key: "Путь к приватному ключу",
    remember_pw: "Запомнить пароль (хранится в системном хранилище, не в файле)",
    btn_connect: "Подключиться",
    install_title: "AmneziaWG ещё не установлен",
    install_sub: "Нажмите «Установить» — всё настроится автоматически. Займёт пару минут.",
    lbl_first_client: "Имя первого клиента",
    advanced: "Дополнительно",
    lbl_port: "UDP-порт (необязательно — оставьте пустым для автоподбора)",
    lbl_profile: "Профиль обфускации",
    profile_mobile: "Мобильный (по умолчанию)",
    profile_desktop: "Десктоп (выше MTU)",
    profile_plain: "Чистый WireGuard (без обфускации)",
    lbl_dns: "DNS клиентов (необязательно)",
    ph_dns1: "1.1.1.1",
    ph_dns2: "1.0.0.1 (резервный)",
    ph_port: "авто",
    lbl_secondary_endpoint: "Резервный Endpoint (необязательно)",
    ph_secondary_endpoint: "146.103.102.101",
    tip_secondary_endpoint: "Второй IP или домен того же сервера. Для каждого клиента будут созданы два одинаковых профиля с разными Endpoint.",
    btn_install: "Установить",
    tab_clients: "Клиенты",
    tab_monitor: "Мониторинг",
    tab_panel: "Веб-панель",
    clients_title: "Клиенты",
    ph_new_client: "имя нового клиента",
    adv_client: "Расширенные настройки клиента",
    f_routes: "Маршруты (AllowedIPs)",
    f_dns: "DNS",
    f_mtu: "MTU",
    f_endpoint_profile: "Профиль Endpoint",
    endpoint_profile_both: "Оба IP: показать два профиля",
    endpoint_profile_primary: "Только основной IP",
    endpoint_profile_secondary: "Только резервный IP",
    hint_advanced: "Пусто = весь трафик (0.0.0.0/0) и серверные DNS/MTU. Split-tunnel: укажите нужные подсети.",
    tip_port: "UDP-порт, на котором сервер слушает VPN. Пусто = подберётся свободный случайный.",
    tip_profile: "Как маскируется трафик от DPI. Мобильный — надёжно везде (4G/ПК). Десктоп — выше MTU для проводных. Чистый WireGuard — без маскировки (быстрее, но цензор видит).",
    tip_dns: "DNS-серверы для клиента. По умолчанию Cloudflare (1.1.1.1 / 1.0.0.1). Можно указать свои через запятую.",
    tip_routes: "Какой трафик клиент шлёт через VPN. Пусто = весь (0.0.0.0/0). Укажите подсети для split-tunnel — тогда через VPN идут только они, остальное напрямую.",
    tip_mtu: "Размер пакета. Пусто = как у сервера. Меньше (напр. 1280) помогает на мобильных сетях, где большие пакеты режутся.",
    btn_add: "Добавить",
    hint_profile: "Один профиль = одно устройство. Для каждого устройства создавайте свой профиль, иначе будет конфликт.",
    th_name: "Имя",
    th_actions: "Действия",
    btn_uninstall: "Удалить AmneziaWG полностью",
    uptime_label: "аптайм",
    stat_status: "Статус",
    stat_clients: "Клиентов",
    stat_uptime: "Аптайм",
    traffic: "Трафик",
    periods_title: "Сводный трафик",
    period_today: "За день",
    period_week: "За неделю",
    period_month: "За месяц",
    period_all: "За всё время",
    periods_hint: "Трафик за период считает веб-панель — у неё есть постоянно работающая служба, которая ведёт учёт. Установите её во вкладке «Веб-панель», чтобы видеть день/неделю/месяц. Ниже — текущий трафик клиентов с момента запуска VPN.",
    th_client: "Клиент",
    th_rx: "↓ принято",
    th_tx: "↑ отдано",
    th_handshake: "рукопожатие",
    panel_title: "Лимиты на клиента",
    panel_desc: "Ограничение скорости, срок действия и квота трафика на каждого клиента настраиваются в веб-панели — у неё для этого есть постоянно работающая на сервере служба. Из приложения это включить нельзя: оно не запущено круглосуточно.",
    ph_panel_pass: "пароль администратора",
    btn_install_panel: "Установить веб-панель",
    panel_pw_rule: "Пароль: минимум 6 символов, строчные и заглавные буквы, цифра и спецсимвол (например <span class=\"mono\">Admin2@</span>).",
    panel_up: "✓ Веб-панель работает:",
    btn_open_panel: "Открыть в браузере",
    btn_remove_panel: "Удалить панель",
    panel_cert_note: "Браузер один раз предупредит про самоподписанный сертификат — это нормально, трафик шифруется.",
    tab_settings: "Настройки",
    tab_terminal: "Терминал сервера",
    terminal_title: "Терминал сервера",
    terminal_hint: "Команды выполняются на сервере по SSH от текущего пользователя. Каждая команда запускается отдельно — рабочая папка и переменные не сохраняются между командами (например, <span class=\"mono\">cd</span> не запомнится).",
    terminal_ph: "например: awg show",
    terminal_run: "Выполнить",
    terminal_clear: "Очистить",
    terminal_suggest: "Полезные команды (нажмите, чтобы выполнить):",
    settings_server: "Сервер",
    info_host: "Адрес",
    info_port: "UDP-порт",
    info_uptime: "Аптайм",
    info_clients: "Клиентов",
    info_panel: "Панель",
    settings_rename: "Название сервера",
    settings_rename_hint: "Понятное имя для этого подключения в приложении. На сам сервер не влияет.",
    ph_server_name: "например, VDSina DE",
    btn_save: "Сохранить",
    settings_secondary_endpoint: "Резервный Endpoint",
    settings_secondary_endpoint_hint: "Второй публичный IP или домен уже настроенного сервера. Сохранение не переустанавливает VPN и не меняет ключи.",
    profile_primary: "Основной IP",
    profile_secondary: "Резервный IP",
    settings_password: "Пароль веб-панели",
    settings_password_hint: "Сменить пароль администратора веб-панели. Применяется сразу.",
    ph_new_panel_pass: "новый пароль",
    btn_change_pass: "Сменить пароль",
    settings_no_panel: "Веб-панель не установлена — её можно установить во вкладке «Веб-панель».",
    settings_danger: "Опасная зона",
    danger_remove_panel_t: "Удалить веб-панель",
    danger_remove_panel_d: "AmneziaWG и клиенты останутся.",
    danger_remove_awg_t: "Удалить AmneziaWG полностью",
    danger_remove_awg_d: "Снесёт VPN, панель, всех клиентов и конфиги с сервера.",
    btn_delete_all: "Удалить всё",
    busy_saving: "Сохраняю…",
    toast_renamed: "Название сохранено",
    e_rename: "Не удалось сохранить название: ",
    e_server_info: "Не удалось получить данные сервера: ",
    log_change_pass: "Смена пароля панели",
    busy_changing_pass: "Меняю пароль…",
    toast_pass_changed: "Пароль панели изменён",
    e_change_pass: "Не удалось сменить пароль: ",
    tab_bot: "Telegram Бот",
    bot_title: "Telegram-бот",
    bot_desc: "Бот выдаёт профили прямо в Telegram: пишешь <span class=\"mono\">/new имя</span> — он присылает .conf и QR. Пользоваться могут только разрешённые Telegram ID, которые ещё и ввели пароль.",
    bot_step1: "<b>Создай бота.</b> Открой <a href=\"#\" id=\"link-botfather\" class=\"mono\">@BotFather</a> в Telegram, отправь <span class=\"mono\">/newbot</span>, пройди шаги и скопируй токен.",
    bot_step2: "<b>Узнай свой Telegram ID.</b> Открой <a href=\"#\" id=\"link-userinfo\" class=\"mono\">@userinfobot</a> — он покажет числовой ID (можно несколько, через запятую).",
    bot_step3: "<b>Придумай пароль доступа.</b> Его будут вводить командой /auth.",
    ph_bot_token: "токен от @BotFather",
    ph_bot_admins: "Telegram ID через запятую",
    ph_bot_pass: "пароль доступа",
    btn_install_bot: "Установить бота",
    btn_remove_bot: "удалить бота",
    bot_up: "✓ Telegram-бот работает.",
    bot_present_hint: "В Telegram (с разрешённого аккаунта): один раз <span class=\"mono\">/auth пароль</span>, затем <span class=\"mono\">/new имя</span>.",
    e_check_bot: "Не удалось проверить бота: ",
    e_bot_token: "Укажи токен бота (от @BotFather).",
    e_bot_admins: "Укажи хотя бы один Telegram ID (от @userinfobot).",
    log_bot: "Установка Telegram-бота",
    busy_installing_bot: "Устанавливаю бота…",
    toast_bot_installed: "Telegram-бот установлен",
    e_install_bot: "Не удалось установить бота: ",
    confirm_remove_bot: "Удалить Telegram-бота с сервера?",
    delete_bot: "Удалить бота",
    busy_removing_bot: "Удаляю бота…",
    toast_bot_removed: "Telegram-бот удалён",
    e_remove_bot: "Не удалось удалить бота: ",
    update_title: "Обновление приложения",
    update_checking: "Проверяю обновления…",
    update_check: "Проверить",
    update_download: "Скачать",
    update_notes: "Что нового",
    update_found: "Доступна {latest} (у вас {current}). Скачайте и установите как обычно.",
    update_uptodate: "У вас последняя версия ✓ ({current})",
    update_failed: "Не удалось проверить обновления.",
    client_ready_prefix: "Клиент",
    client_ready_suffix: "готов",
    qr_note: "Откройте приложение <b>AmneziaWG</b> и отсканируйте QR — или импортируйте файл .conf.",
    download_awg: "Скачать AmneziaWG:",
    btn_download_conf: "Скачать .conf",
    cancel: "Отмена",

    act_config: "конфиг / QR",
    act_limits: "лимиты",
    act_rename: "переименовать",
    act_delete: "удалить",
    chip_disabled: "отключён",
    unit_mbit: "Мбит",
    unit_gb: "ГБ",
    unit_day: "д",
    limits_title: "Лимиты клиента",
    limits_hint: "Ограничения применяет фоновая служба веб-панели. Пусто = без ограничения.",
    lim_speed: "Скорость, Мбит/с",
    lim_quota: "Квота, ГБ",
    lim_expiry: "Срок, дней",
    lim_enabled: "Клиент включён",
    lim_used: "Использовано: {used}",
    toast_limits_saved: "Лимиты сохранены",
    e_limits: "Не удалось сохранить лимиты: ",
    delete: "Удалить",
    remove: "Убрать",
    delete_all: "Удалить всё",
    delete_panel: "Удалить панель",
    profile_remove_title: "Убрать из списка",
    clients_empty: "Пока нет клиентов. Добавьте первого выше.",
    traffic_empty: "Пока нет данных",
    vpn_running: "VPN работает",
    vpn_stopped: "VPN остановлен",
    status_unavailable: "статус недоступен",
    clients_count: "{n} клиент(ов)",
    traffic_summary: "(онлайн {online} из {total})",

    log_install: "Установка",
    log_uninstall: "Удаление",
    log_panel: "Установка веб-панели",

    busy_default: "Подождите…",
    busy_connecting: "Подключаюсь к серверу…",
    busy_checking: "Проверяю сервер…",
    busy_installing: "Устанавливаю AmneziaWG… это займёт пару минут",
    busy_creating: "Создаю клиента…",
    busy_loading_conf: "Загружаю конфиг…",
    busy_renaming: "Переименовываю…",
    busy_deleting_client: "Удаляю клиента…",
    busy_uninstalling: "Удаляю AmneziaWG…",
    busy_installing_panel: "Устанавливаю веб-панель…",
    busy_removing_panel: "Удаляю панель…",

    toast_installed: "AmneziaWG установлен",
    toast_uninstalled: "AmneziaWG полностью удалён",
    toast_panel_installed: "Веб-панель установлена",
    toast_panel_removed: "Веб-панель удалена",
    toast_saved: "Сохранено: {path}",
    toast_client_removed: "Клиент «{name}» удалён",
    toast_client_renamed: "Клиент переименован в «{name}»",

    err_no_host: "Укажите IP-адрес сервера",
    err_weak_pw: "Слабый пароль: мин. 6 символов, строчные и заглавные буквы, цифра и спецсимвол (например Admin2@)",
    e_check_server: "Не удалось проверить сервер: ",
    e_install: "Установка не удалась: ",
    e_create_client: "Не удалось создать клиента: ",
    e_load_conf: "Не удалось получить конфиг: ",
    e_rename: "Не удалось переименовать: ",
    e_delete_client: "Не удалось удалить клиента: ",
    e_uninstall: "Не удалось удалить: ",
    e_check_panel: "Не удалось проверить панель: ",
    e_install_panel: "Не удалось установить панель: ",
    e_remove_panel: "Не удалось удалить панель: ",
    e_open_browser: "Не удалось открыть браузер: ",
    e_save: "Не удалось сохранить: ",
    e_secondary_endpoint: "Не удалось сохранить резервный Endpoint: ",

    confirm_remove_client: "Удалить клиента «{name}»? Его профиль перестанет работать.",
    confirm_uninstall: "Это ПОЛНОСТЬЮ удалит AmneziaWG, веб-панель, всех клиентов и конфиги с сервера. Продолжить?",
    confirm_remove_panel: "Удалить веб-панель с сервера? AmneziaWG и клиенты останутся.",
    confirm_remove_profile: "Убрать сервер «{name}» из списка? Сам сервер не изменится.",
    prompt_rename: "Новое имя для клиента «{name}»:",
  },
  en: {
    switch_server: "Switch server",
    connect_title: "Connect to a server",
    connect_sub: "Enter your Linux server details. The app connects over SSH and does the rest.",
    howto_summary: "How it works (3 steps)",
    howto_1: "<b>Server.</b> You need any Ubuntu/Debian VPS (from a hosting provider) — its IP, a user (usually <span class=\"mono\">root</span>) and the password.",
    howto_2: "<b>Install.</b> Connect here — the app installs AmneziaWG and creates the first client for you.",
    howto_3: "<b>Connect.</b> Install the AmneziaWG app on your phone/PC and scan the QR (or import the .conf file).",
    rent_lead: "No server yet? You can rent a VPS:",
    rent_note: "(referral links — they support the project)",
    saved_servers: "Saved servers",
    lbl_host: "Server IP address",
    lbl_user: "User",
    lbl_label: "Label (optional)",
    ph_label: "e.g. VDSina, Germany",
    auth_password: "Password",
    auth_key: "SSH key",
    lbl_password: "Password",
    lbl_key: "Private key path",
    remember_pw: "Remember password (stored in the OS keychain, not in a file)",
    btn_connect: "Connect",
    install_title: "AmneziaWG is not installed yet",
    install_sub: "Click Install — everything is configured automatically. Takes a couple of minutes.",
    lbl_first_client: "First client name",
    advanced: "Advanced",
    lbl_port: "UDP port (optional — leave empty for auto)",
    lbl_profile: "Obfuscation profile",
    profile_mobile: "Mobile (default)",
    profile_desktop: "Desktop (higher MTU)",
    profile_plain: "Plain WireGuard (no obfuscation)",
    lbl_dns: "Client DNS (optional)",
    ph_dns1: "1.1.1.1",
    ph_dns2: "1.0.0.1 (secondary)",
    ph_port: "auto",
    lbl_secondary_endpoint: "Fallback Endpoint (optional)",
    ph_secondary_endpoint: "146.103.102.101",
    tip_secondary_endpoint: "A second IP or hostname for the same server. Each client gets two otherwise identical profiles with different Endpoints.",
    btn_install: "Install",
    tab_clients: "Clients",
    tab_monitor: "Monitoring",
    tab_panel: "Web panel",
    clients_title: "Clients",
    ph_new_client: "new client name",
    adv_client: "Advanced client settings",
    f_routes: "Routes (AllowedIPs)",
    f_dns: "DNS",
    f_mtu: "MTU",
    f_endpoint_profile: "Endpoint profile",
    endpoint_profile_both: "Both IPs: show two profiles",
    endpoint_profile_primary: "Primary IP only",
    endpoint_profile_secondary: "Fallback IP only",
    hint_advanced: "Empty = all traffic (0.0.0.0/0) and the server's DNS/MTU. Split tunnel: list the subnets to route.",
    tip_port: "The UDP port the server listens on for the VPN. Empty = auto-pick a free random one.",
    tip_profile: "How traffic is disguised from DPI. Mobile — reliable everywhere (4G/PC). Desktop — higher MTU for wired links. Plain WireGuard — no disguise (faster, but visible to censors).",
    tip_dns: "DNS servers the client uses. Default is Cloudflare (1.1.1.1 / 1.0.0.1). You can set your own, comma-separated.",
    tip_routes: "Which traffic the client sends through the VPN. Empty = all (0.0.0.0/0). List subnets for split tunnel — then only those go through the VPN, the rest stays direct.",
    tip_mtu: "Packet size. Empty = same as the server. Smaller (e.g. 1280) helps on mobile networks that drop large packets.",
    btn_add: "Add",
    hint_profile: "One profile = one device. Create a separate profile per device, or connections will clash.",
    th_name: "Name",
    th_actions: "Actions",
    btn_uninstall: "Remove AmneziaWG completely",
    uptime_label: "uptime",
    stat_status: "Status",
    stat_clients: "Clients",
    stat_uptime: "Uptime",
    traffic: "Traffic",
    periods_title: "Traffic summary",
    period_today: "Today",
    period_week: "Last 7 days",
    period_month: "Last 30 days",
    period_all: "All time",
    periods_hint: "Per-period traffic is tracked by the web panel — it runs an always-on service that records usage. Install it on the \"Web panel\" tab to see day/week/month. Below is the live client traffic since the VPN last started.",
    th_client: "Client",
    th_rx: "↓ received",
    th_tx: "↑ sent",
    th_handshake: "handshake",
    panel_title: "Per-client limits",
    panel_desc: "Speed limit, expiry and traffic quota per client are configured in the web panel — it runs an always-on service on the server for that. The desktop app can't do it: it isn't running 24/7.",
    ph_panel_pass: "admin password",
    btn_install_panel: "Install web panel",
    panel_pw_rule: "Password: at least 6 chars with lower- and upper-case letters, a digit and a special character (e.g. <span class=\"mono\">Admin2@</span>).",
    panel_up: "✓ Web panel is running:",
    btn_open_panel: "Open in browser",
    btn_remove_panel: "Remove panel",
    panel_cert_note: "The browser warns once about the self-signed certificate — that's expected; traffic is still encrypted.",
    tab_settings: "Settings",
    tab_terminal: "Server terminal",
    terminal_title: "Server terminal",
    terminal_hint: "Commands run on the server over SSH as the current user. Each command runs on its own — the working directory and variables aren't kept between commands (e.g. <span class=\"mono\">cd</span> won't persist).",
    terminal_ph: "e.g. awg show",
    terminal_run: "Run",
    terminal_clear: "Clear",
    terminal_suggest: "Useful commands (click to run):",
    settings_server: "Server",
    info_host: "Address",
    info_port: "UDP port",
    info_uptime: "Uptime",
    info_clients: "Clients",
    info_panel: "Panel",
    settings_rename: "Server name",
    settings_rename_hint: "A friendly name for this connection in the app. Does not affect the server itself.",
    ph_server_name: "e.g. VDSina DE",
    btn_save: "Save",
    settings_secondary_endpoint: "Fallback Endpoint",
    settings_secondary_endpoint_hint: "A second public IP or hostname already routed to this server. Saving does not reinstall the VPN or change keys.",
    profile_primary: "Primary IP",
    profile_secondary: "Fallback IP",
    settings_password: "Web panel password",
    settings_password_hint: "Change the web panel admin password. Applies immediately.",
    ph_new_panel_pass: "new password",
    btn_change_pass: "Change password",
    settings_no_panel: "The web panel is not installed — you can install it on the \"Web panel\" tab.",
    settings_danger: "Danger zone",
    danger_remove_panel_t: "Remove the web panel",
    danger_remove_panel_d: "AmneziaWG and clients stay.",
    danger_remove_awg_t: "Remove AmneziaWG completely",
    danger_remove_awg_d: "Wipes the VPN, panel, all clients and configs from the server.",
    btn_delete_all: "Delete all",
    busy_saving: "Saving…",
    toast_renamed: "Name saved",
    e_rename: "Could not save the name: ",
    e_server_info: "Could not fetch server info: ",
    log_change_pass: "Changing panel password",
    busy_changing_pass: "Changing password…",
    toast_pass_changed: "Panel password changed",
    e_change_pass: "Could not change the password: ",
    tab_bot: "Telegram Bot",
    bot_title: "Telegram bot",
    bot_desc: "The bot hands out profiles right in Telegram: send <span class=\"mono\">/new name</span> and it replies with the .conf + QR. Only allowlisted Telegram IDs that also enter the password can use it.",
    bot_step1: "<b>Create a bot.</b> Open <a href=\"#\" id=\"link-botfather\" class=\"mono\">@BotFather</a> in Telegram, send <span class=\"mono\">/newbot</span>, follow the steps and copy the token.",
    bot_step2: "<b>Find your Telegram ID.</b> Open <a href=\"#\" id=\"link-userinfo\" class=\"mono\">@userinfobot</a> — it shows your numeric ID (you can list several, comma-separated).",
    bot_step3: "<b>Choose an access password.</b> Users enter it with /auth.",
    ph_bot_token: "token from @BotFather",
    ph_bot_admins: "Telegram IDs, comma-separated",
    ph_bot_pass: "access password",
    btn_install_bot: "Install the bot",
    btn_remove_bot: "remove bot",
    bot_up: "✓ The Telegram bot is running.",
    bot_present_hint: "In Telegram (from an allowlisted account): once <span class=\"mono\">/auth password</span>, then <span class=\"mono\">/new name</span>.",
    e_check_bot: "Could not check the bot: ",
    e_bot_token: "Enter the bot token (from @BotFather).",
    e_bot_admins: "Enter at least one Telegram ID (from @userinfobot).",
    log_bot: "Installing the Telegram bot",
    busy_installing_bot: "Installing the bot…",
    toast_bot_installed: "Telegram bot installed",
    e_install_bot: "Could not install the bot: ",
    confirm_remove_bot: "Remove the Telegram bot from the server?",
    delete_bot: "Remove bot",
    busy_removing_bot: "Removing the bot…",
    toast_bot_removed: "Telegram bot removed",
    e_remove_bot: "Could not remove the bot: ",
    update_title: "App update",
    update_checking: "Checking for updates…",
    update_check: "Check",
    update_download: "Download",
    update_notes: "What's new",
    update_found: "{latest} is available (you have {current}). Download and install as usual.",
    update_uptodate: "You're on the latest version ✓ ({current})",
    update_failed: "Couldn't check for updates.",
    client_ready_prefix: "Client",
    client_ready_suffix: "is ready",
    qr_note: "Open the <b>AmneziaWG</b> app and scan the QR — or import the .conf file.",
    download_awg: "Download AmneziaWG:",
    btn_download_conf: "Download .conf",
    cancel: "Cancel",

    act_config: "config / QR",
    act_limits: "limits",
    act_rename: "rename",
    act_delete: "delete",
    chip_disabled: "disabled",
    unit_mbit: "Mbit",
    unit_gb: "GB",
    unit_day: "d",
    limits_title: "Client limits",
    limits_hint: "Limits are enforced by the web panel's background service. Empty = no limit.",
    lim_speed: "Speed, Mbit/s",
    lim_quota: "Quota, GB",
    lim_expiry: "Expiry, days",
    lim_enabled: "Client enabled",
    lim_used: "Used: {used}",
    toast_limits_saved: "Limits saved",
    e_limits: "Could not save limits: ",
    delete: "Delete",
    remove: "Remove",
    delete_all: "Delete everything",
    delete_panel: "Remove panel",
    profile_remove_title: "Remove from list",
    clients_empty: "No clients yet. Add the first one above.",
    traffic_empty: "No data yet",
    vpn_running: "VPN is running",
    vpn_stopped: "VPN is stopped",
    status_unavailable: "status unavailable",
    clients_count: "{n} client(s)",
    traffic_summary: "({online} of {total} online)",

    log_install: "Installation",
    log_uninstall: "Removal",
    log_panel: "Web panel install",

    busy_default: "Please wait…",
    busy_connecting: "Connecting to the server…",
    busy_checking: "Checking the server…",
    busy_installing: "Installing AmneziaWG… this takes a couple of minutes",
    busy_creating: "Creating client…",
    busy_loading_conf: "Loading config…",
    busy_renaming: "Renaming…",
    busy_deleting_client: "Removing client…",
    busy_uninstalling: "Removing AmneziaWG…",
    busy_installing_panel: "Installing web panel…",
    busy_removing_panel: "Removing panel…",

    toast_installed: "AmneziaWG installed",
    toast_uninstalled: "AmneziaWG fully removed",
    toast_panel_installed: "Web panel installed",
    toast_panel_removed: "Web panel removed",
    toast_saved: "Saved: {path}",
    toast_client_removed: "Client \"{name}\" removed",
    toast_client_renamed: "Client renamed to \"{name}\"",

    err_no_host: "Enter the server IP address",
    err_weak_pw: "Weak password: at least 6 chars with lower- and upper-case letters, a digit and a special character (e.g. Admin2@)",
    e_check_server: "Could not check the server: ",
    e_install: "Install failed: ",
    e_create_client: "Could not create the client: ",
    e_load_conf: "Could not get the config: ",
    e_rename: "Could not rename: ",
    e_delete_client: "Could not remove the client: ",
    e_uninstall: "Could not remove: ",
    e_check_panel: "Could not check the panel: ",
    e_install_panel: "Could not install the panel: ",
    e_remove_panel: "Could not remove the panel: ",
    e_open_browser: "Could not open the browser: ",
    e_save: "Could not save: ",
    e_secondary_endpoint: "Could not save fallback Endpoint: ",

    confirm_remove_client: "Remove client \"{name}\"? Its profile will stop working.",
    confirm_uninstall: "This will COMPLETELY remove AmneziaWG, the web panel, all clients and configs from the server. Continue?",
    confirm_remove_panel: "Remove the web panel from the server? AmneziaWG and clients stay.",
    confirm_remove_profile: "Remove server \"{name}\" from the list? The server itself is not changed.",
    prompt_rename: "New name for client \"{name}\":",
  },
};

let LANG = localStorage.getItem("awg-lang") === "en" ? "en" : "ru";

function t(key, vars) {
  let s = (I18N[LANG] && I18N[LANG][key]) || I18N.ru[key] || key;
  if (vars) for (const k in vars) s = s.replaceAll("{" + k + "}", vars[k]);
  return s;
}

function applyI18n() {
  document.documentElement.lang = LANG;
  document.querySelectorAll("[data-i18n]").forEach((el) => { el.textContent = t(el.dataset.i18n); });
  document.querySelectorAll("[data-i18n-html]").forEach((el) => { el.innerHTML = t(el.dataset.i18nHtml); });
  document.querySelectorAll("[data-i18n-ph]").forEach((el) => { el.placeholder = t(el.dataset.i18nPh); });
  document.querySelectorAll(".langswitch a").forEach((a) => a.classList.toggle("on", a.dataset.lang === LANG));
}

async function setLang(lang) {
  LANG = lang === "en" ? "en" : "ru";
  localStorage.setItem("awg-lang", LANG);
  try { await backend().SetLang(LANG); } catch (_) { /* ignore */ }
  applyI18n();
  // Re-render dynamic text that isn't covered by data-i18n.
  if (!$("view-connect").classList.contains("hidden")) loadProfiles();
  if (!$("view-server").classList.contains("hidden") && !$("block-manage").classList.contains("hidden")) {
    refreshClients();
    refreshHealth();
    refreshTraffic();
  }
}

// openExternal opens a URL in the user's real browser, not the app webview.
function openExternal(url) {
  if (window.runtime && window.runtime.BrowserOpenURL) window.runtime.BrowserOpenURL(url);
}

let authMode = "password";
let lastResult = null;
let panelInstalled = false; // whether the web panel (and its enforcing daemon) is installed
let clientLimits = {}; // name -> ClientLimit, populated when the panel is installed
let limitsTarget = null; // client name currently open in the limits modal

// --- small UI helpers ------------------------------------------------------

function show(el) { el.classList.remove("hidden"); }
function hide(el) { el.classList.add("hidden"); }

function busy(on, text) {
  $("busy-text").textContent = text || t("busy_default");
  on ? show($("busy")) : hide($("busy"));
}

// confirmDialog shows an in-app modal and resolves true/false. Native confirm()
// is unreliable inside the Wails webview, so we never use it.
function confirmDialog(message, okLabel) {
  return new Promise((resolve) => {
    $("modal-text").textContent = message;
    $("modal-ok").textContent = okLabel || t("delete");
    show($("modal"));
    const done = (val) => {
      hide($("modal"));
      $("modal-ok").onclick = null;
      $("modal-cancel").onclick = null;
      resolve(val);
    };
    $("modal-ok").onclick = () => done(true);
    $("modal-cancel").onclick = () => done(false);
  });
}

let toastTimer = null;
function toast(message, kind = "ok") {
  const el = $("toast");
  el.textContent = message;
  el.className = "toast " + kind;
  show(el);
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => hide(el), 4500);
}

function errMsg(err) {
  return typeof err === "string" ? err : (err && err.message) || String(err);
}

// openLog clears and reveals the floating log drawer for an operation.
function openLog(title) {
  $("log").textContent = "";
  $("log-title").textContent = title;
  const d = $("log-panel");
  d.classList.remove("hidden");
  d.open = true;
}

// closeLog dismisses the floating log drawer.
function closeLog() {
  hide($("log-panel"));
}

// promptDialog shows an in-app text-input modal and resolves the value (or null).
function promptDialog(message, defaultValue = "") {
  return new Promise((resolve) => {
    $("prompt-text").textContent = message;
    const input = $("prompt-input");
    input.value = defaultValue;
    show($("prompt"));
    input.focus();
    input.select();
    const done = (val) => {
      hide($("prompt"));
      $("prompt-ok").onclick = null;
      input.onkeydown = null;
      resolve(val);
    };
    $("prompt-ok").onclick = () => done(input.value.trim());
    input.onkeydown = (e) => { if (e.key === "Enter") done(input.value.trim()); };
    window.__closePrompt = () => done(null);
  });
}

function closePrompt() {
  if (window.__closePrompt) window.__closePrompt();
}

// switchServer disconnects and returns to the connect screen to pick/add a server.
async function switchServer() {
  stopTraffic();
  try {
    await backend().Disconnect();
  } catch (_) {
    /* ignore */
  }
  hide($("view-server"));
  hide($("conn-pill"));
  hide($("btn-switch"));
  show($("view-connect"));
  $("password").value = "";
  await prefill();
}

function showError(el, err) {
  el.textContent = typeof err === "string" ? err : (err && err.message) || String(err);
  show(el);
}

function appendLog(text) {
  const log = $("log");
  log.textContent += text;
  log.scrollTop = log.scrollHeight;
}

// --- connect ---------------------------------------------------------------

function selectAuth(mode) {
  authMode = mode;
  document.querySelectorAll(".tab").forEach((tab) => tab.classList.toggle("on", tab.dataset.auth === mode));
  $("field-password").classList.toggle("hidden", mode !== "password");
  $("field-key").classList.toggle("hidden", mode !== "key");
}

function initAuthTabs() {
  document.querySelectorAll(".tab").forEach((tab) => {
    tab.addEventListener("click", () => selectAuth(tab.dataset.auth));
  });
}

const MANAGE_TABS = ["clients", "monitor", "advanced", "bot", "settings", "terminal"];

function selectTab(name) {
  document.querySelectorAll(".nav-item").forEach((b) => b.classList.toggle("on", b.dataset.tab === name));
  MANAGE_TABS.forEach((tab) => $("tab-" + tab).classList.toggle("hidden", tab !== name));
  hide($("log-panel")); // the floating op-log shouldn't linger when changing tabs
  if (name === "settings") loadSettings();
  if (name === "bot") refreshBot();
  if (name === "monitor") refreshPeriods();
  if (name === "terminal") setTimeout(() => $("term-cmd").focus(), 0);
}

function initTabs() {
  document.querySelectorAll(".nav-item").forEach((btn) => {
    btn.addEventListener("click", () => selectTab(btn.dataset.tab));
  });
}

// profileLabel is the display name for a saved server: its label or user@host.
function profileLabel(p) {
  return (p.label && p.label.trim()) || (p.user || "root") + "@" + p.host;
}

// fillForm populates the connect form from a saved Prefs/Profile object.
function fillForm(p) {
  if (!p) return;
  $("host").value = p.host || "";
  $("user").value = p.user || "root";
  $("srv-label").value = p.label || "";
  $("identity").value = p.identityPath || "";
  selectAuth(p.authMode || "password");
  $("remember").checked = !!p.remember;
  $("password").value = p.password || "";
}

// prefill restores the last-used connection (password from the OS secret store)
// and renders the saved-servers list.
async function prefill() {
  try {
    fillForm(await backend().LoadPrefs());
  } catch (_) {
    /* no saved prefs yet — ignore */
  }
  await loadProfiles();
}

// loadProfiles renders saved servers as one-click reconnect rows.
async function loadProfiles() {
  let list = [];
  try {
    list = (await backend().ListProfiles()) || [];
  } catch (_) {
    return;
  }
  const box = $("profiles");
  const ul = $("profiles-list");
  ul.innerHTML = "";
  if (list.length === 0) {
    hide(box);
    return;
  }
  list.forEach((p) => {
    const label = profileLabel(p);
    const row = document.createElement("div");
    row.className = "profile";

    const pick = document.createElement("button");
    pick.className = "pick";
    pick.textContent = label;
    pick.addEventListener("click", () => {
      fillForm(p);
      connect();
    });

    const x = document.createElement("button");
    x.className = "x";
    x.textContent = "×";
    x.title = t("profile_remove_title");
    x.addEventListener("click", async (e) => {
      e.stopPropagation();
      const ok = await confirmDialog(t("confirm_remove_profile", { name: label }), t("remove"));
      if (!ok) return;
      try {
        await backend().DeleteProfile(p.host, p.user);
      } catch (_) {
        /* ignore */
      }
      await loadProfiles();
    });

    row.append(pick, x);
    ul.appendChild(row);
  });
  show(box);
}

async function connect() {
  hide($("connect-err"));
  const host = $("host").value.trim();
  if (!host) { showError($("connect-err"), t("err_no_host")); return; }

  const user = $("user").value.trim() || "root";
  const label = $("srv-label").value.trim();
  const req = {
    host,
    user,
    label,
    password: authMode === "password" ? $("password").value : "",
    identityPath: authMode === "key" ? $("identity").value.trim() : "",
    authMode,
    remember: $("remember").checked,
  };

  busy(true, t("busy_connecting"));
  try {
    await backend().Connect(req);
    $("conn-pill").textContent = label || user + "@" + host;
    show($("conn-pill"));
    show($("btn-switch"));
    hide($("view-connect"));
    show($("view-server"));
    await refreshStatus();
    loadProfiles();
  } catch (err) {
    showError($("connect-err"), err);
  } finally {
    busy(false);
  }
}

// --- server status: install vs manage --------------------------------------

async function refreshStatus() {
  busy(true, t("busy_checking"));
  try {
    const status = await backend().ServerStatus();
    hide($("result"));
    if (status.installed) {
      hide($("block-install"));
      show($("block-manage"));
      selectTab("clients");
      await refreshPanel(); // sets panelInstalled before clients render their limit controls
      await refreshClients();
      await refreshHealth();
      startTraffic();
    } else {
      show($("block-install"));
      hide($("block-manage"));
      stopTraffic();
    }
  } catch (err) {
    toast(t("e_check_server") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

// --- install ---------------------------------------------------------------

async function install() {
  const req = {
    port: $("port").value.trim(),
    client: $("first-client").value.trim() || "phone",
    preset: $("obf-profile").value,
    dns1: $("dns1").value.trim(),
    dns2: $("dns2").value.trim(),
    secondaryEndpoint: $("secondary-endpoint").value.trim(),
  };
  openLog(t("log_install"));
  $("btn-install").disabled = true;
  busy(true, t("busy_installing"));
  try {
    const res = await backend().Install(req);
    await refreshStatus();
    showResult(res);
    toast(t("toast_installed"), "ok");
  } catch (err) {
    toast(t("e_install") + errMsg(err), "err");
  } finally {
    busy(false);
    $("btn-install").disabled = false;
  }
}

// --- manage clients --------------------------------------------------------

async function refreshClients() {
  const names = (await backend().ListClients()) || [];
  // When the panel is installed it owns the enforcing daemon, so per-client
  // limits (speed/quota/expiry) can be managed from here too.
  clientLimits = {};
  if (panelInstalled) {
    try {
      const limits = (await backend().ClientLimits()) || [];
      limits.forEach((l) => { clientLimits[l.name] = l; });
    } catch (_) { /* leave limits empty; controls just won't show */ }
  }
  const body = $("clients-body");
  body.innerHTML = "";
  if (names.length === 0) {
    const tr = document.createElement("tr");
    const td = document.createElement("td");
    td.colSpan = 2;
    td.className = "empty";
    td.textContent = t("clients_empty");
    tr.appendChild(td);
    body.appendChild(tr);
    return;
  }
  names.forEach((name) => {
    const tr = document.createElement("tr");
    const nameTd = document.createElement("td");
    nameTd.className = "name";
    nameTd.appendChild(document.createTextNode(name));
    const chip = limitChip(clientLimits[name]);
    if (chip) nameTd.appendChild(chip);

    const actTd = document.createElement("td");
    actTd.className = "r";
    actTd.innerHTML = '<div class="row-actions"></div>';
    const actions = actTd.querySelector(".row-actions");

    const conf = document.createElement("button");
    conf.className = "link";
    conf.textContent = t("act_config");
    conf.addEventListener("click", () => showClientConfig(name));
    actions.appendChild(conf);

    if (panelInstalled) {
      const lim = document.createElement("button");
      lim.className = "link";
      lim.textContent = t("act_limits");
      lim.addEventListener("click", () => openLimits(name));
      actions.appendChild(lim);
    }

    const ren = document.createElement("button");
    ren.className = "link";
    ren.textContent = t("act_rename");
    ren.addEventListener("click", () => renameClient(name));
    actions.appendChild(ren);

    const del = document.createElement("button");
    del.className = "link del";
    del.textContent = t("act_delete");
    del.addEventListener("click", () => removeClient(name));
    actions.appendChild(del);

    tr.append(nameTd, actTd);
    body.appendChild(tr);
  });
}

// limitChip builds a small badge summarizing a client's limits, or null if the
// client has none and is enabled. Disabled clients always get a chip.
function limitChip(l) {
  if (!l) return null;
  if (l.disabled) {
    const c = document.createElement("span");
    c.className = "chip chip-off";
    c.textContent = t("chip_disabled");
    return c;
  }
  const parts = [];
  if (l.speedMbit > 0) parts.push(l.speedMbit + " " + t("unit_mbit"));
  if (l.quotaGB > 0) parts.push(l.quotaGB + " " + t("unit_gb"));
  if (l.expiresDays > 0) parts.push(l.expiresDays + t("unit_day"));
  if (parts.length === 0) return null;
  const c = document.createElement("span");
  c.className = "chip";
  c.textContent = parts.join(" · ");
  return c;
}

// openLimits fills and shows the per-client limits modal.
function openLimits(name) {
  const l = clientLimits[name] || { quotaGB: 0, speedMbit: 0, expiresDays: 0, disabled: false, usedBytes: 0 };
  limitsTarget = name;
  $("limits-name").textContent = name;
  $("lim-speed").value = l.speedMbit > 0 ? l.speedMbit : "";
  $("lim-quota").value = l.quotaGB > 0 ? l.quotaGB : "";
  $("lim-expiry").value = l.expiresDays > 0 ? l.expiresDays : "";
  $("lim-enabled").checked = !l.disabled;
  $("lim-used").textContent = t("lim_used", { used: humanBytes(l.usedBytes) });
  show($("limits-modal"));
  $("lim-speed").focus();
}

async function saveLimits() {
  const name = limitsTarget;
  if (!name) return;
  const quota = Math.max(0, parseInt($("lim-quota").value, 10) || 0);
  const speed = Math.max(0, parseInt($("lim-speed").value, 10) || 0);
  const expiry = Math.max(0, parseInt($("lim-expiry").value, 10) || 0);
  const enabled = $("lim-enabled").checked;
  const cur = clientLimits[name];
  const wasEnabled = cur ? !cur.disabled : true;
  busy(true, t("busy_saving"));
  try {
    await backend().SetClientLimits(name, quota, expiry, speed);
    if (enabled !== wasEnabled) await backend().SetClientEnabled(name, enabled);
    hide($("limits-modal"));
    limitsTarget = null;
    await refreshClients();
    toast(t("toast_limits_saved"), "ok");
  } catch (err) {
    toast(t("e_limits") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

// humanBytes formats a byte count compactly (GUI-side, for the usage line).
function humanBytes(n) {
  n = Number(n) || 0;
  const u = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
  return (i === 0 ? n : n.toFixed(1)) + " " + u[i];
}

async function addClient(e) {
  e.preventDefault();
  const name = $("new-client").value.trim();
  if (!name) return;
  busy(true, t("busy_creating"));
  try {
    const res = await backend().AddClient(
      name,
      $("adv-allowed").value.trim(),
      $("adv-dns").value.trim(),
      $("adv-mtu").value.trim(),
      $("adv-endpoint-profile").value,
    );
    $("new-client").value = "";
    ["adv-allowed", "adv-dns", "adv-mtu"].forEach((id) => { $(id).value = ""; });
    $("adv-endpoint-profile").value = "both";
    await refreshClients();
    showResult(res);
  } catch (err) {
    toast(t("e_create_client") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function showClientConfig(name) {
  busy(true, t("busy_loading_conf"));
  try {
    const res = await backend().ClientConfig(name);
    showResult(res);
  } catch (err) {
    toast(t("e_load_conf") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function renameClient(name) {
  const newName = await promptDialog(t("prompt_rename", { name }), name);
  if (!newName || newName === name) return;
  busy(true, t("busy_renaming"));
  try {
    await backend().RenameClient(name, newName);
    await refreshClients();
    toast(t("toast_client_renamed", { name: newName }), "ok");
  } catch (err) {
    toast(t("e_rename") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function removeClient(name) {
  const ok = await confirmDialog(t("confirm_remove_client", { name }));
  if (!ok) return;
  busy(true, t("busy_deleting_client"));
  try {
    await backend().RemoveClient(name);
    await refreshClients();
    toast(t("toast_client_removed", { name }), "ok");
  } catch (err) {
    toast(t("e_delete_client") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function uninstall() {
  const ok = await confirmDialog(t("confirm_uninstall"), t("delete_all"));
  if (!ok) return;
  openLog(t("log_uninstall"));
  busy(true, t("busy_uninstalling"));
  try {
    await backend().Uninstall();
    await refreshStatus();
    toast(t("toast_uninstalled"), "ok");
  } catch (err) {
    toast(t("e_uninstall") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

// --- server health & live traffic ------------------------------------------

async function refreshHealth() {
  try {
    const h = await backend().ServerHealth();
    $("health-dot").className = "health-dot " + (h.running ? "up" : "down");
    $("health-state").textContent = h.running ? t("vpn_running") : t("vpn_stopped");
    $("health-clients").textContent = h.clients;
    $("health-uptime").textContent = h.uptime || "—";
    $("health-version").textContent = h.version || "—";
  } catch (_) {
    $("health-dot").className = "health-dot";
    $("health-state").textContent = t("status_unavailable");
  }
}

// refreshPeriods fetches aggregate traffic over day/week/month. The windows need
// the web panel's always-on sampler; without it we show only a hint + all-time.
async function refreshPeriods() {
  const row = $("periods-row");
  const hint = $("periods-hint");
  try {
    const p = await backend().TrafficPeriods();
    if (p.tracked) {
      $("period-today").textContent = p.today || "—";
      $("period-week").textContent = p.week || "—";
      $("period-month").textContent = p.month || "—";
      $("period-all").textContent = p.allTime || "—";
      show(row);
      hide(hint);
    } else {
      hide(row);
      show(hint);
    }
  } catch (_) {
    hide(row);
    hide(hint);
  }
}

let trafficTimer = null;
let trafficBusy = false;
let trafficPeers = []; // last fetched peers, re-sorted client-side
// trafficSort persists across the 5s poll so the chosen order doesn't reset.
const trafficSort = { key: "name", dir: 1 };

// sortPeers orders a copy of the peers by the active column/direction.
function sortPeers(peers) {
  const { key, dir } = trafficSort;
  const val = (p) => {
    switch (key) {
      case "rx": return p.rxBytes || 0;
      case "tx": return p.txBytes || 0;
      case "hs": return p.handshakeUnix || 0;
      default: return null; // name → string compare below
    }
  };
  return peers.slice().sort((a, b) => {
    if (key === "name") return dir * a.name.localeCompare(b.name);
    return dir * ((val(a) - val(b)) || a.name.localeCompare(b.name));
  });
}

// renderTraffic paints the table body from trafficPeers using the active sort.
function renderTraffic() {
  const body = $("traffic-body");
  body.innerHTML = "";
  if (!trafficPeers || trafficPeers.length === 0) {
    const tr = document.createElement("tr");
    const td = document.createElement("td");
    td.colSpan = 5;
    td.className = "traffic-empty";
    td.textContent = t("traffic_empty");
    tr.appendChild(td);
    body.appendChild(tr);
    updateSortIndicators();
    return;
  }
  sortPeers(trafficPeers).forEach((p) => {
    const tr = document.createElement("tr");
    const dotTd = document.createElement("td");
    dotTd.innerHTML = `<span class="tdot${p.online ? " on" : ""}"></span>`;
    const name = document.createElement("td");
    name.className = "name";
    name.textContent = p.name;
    const rx = document.createElement("td");
    rx.className = "r mono";
    rx.textContent = p.rx;
    const tx = document.createElement("td");
    tx.className = "r mono";
    tx.textContent = p.tx;
    const hs = document.createElement("td");
    hs.className = "r dim";
    hs.textContent = p.handshake;
    tr.append(dotTd, name, rx, tx, hs);
    body.appendChild(tr);
  });
  updateSortIndicators();
}

// updateSortIndicators reflects the active column/direction in the headers.
function updateSortIndicators() {
  document.querySelectorAll("#tab-monitor th.sort").forEach((th) => {
    const ind = th.querySelector(".sort-ind");
    if (th.dataset.sort === trafficSort.key) {
      th.classList.add("on");
      if (ind) ind.textContent = trafficSort.dir > 0 ? " ▲" : " ▼";
    } else {
      th.classList.remove("on");
      if (ind) ind.textContent = "";
    }
  });
}

// initTrafficSort wires header clicks: same column toggles direction, a new one
// sorts ascending (descending first for the numeric byte/handshake columns).
function initTrafficSort() {
  document.querySelectorAll("#tab-monitor th.sort").forEach((th) => {
    th.addEventListener("click", () => {
      const key = th.dataset.sort;
      if (trafficSort.key === key) {
        trafficSort.dir = -trafficSort.dir;
      } else {
        trafficSort.key = key;
        trafficSort.dir = key === "name" ? 1 : -1;
      }
      renderTraffic();
    });
  });
}

async function refreshTraffic() {
  if (trafficBusy) return;
  trafficBusy = true;
  try {
    const r = await backend().Traffic();
    $("traffic-summary").textContent = t("traffic_summary", { online: r.online, total: r.total });
    trafficPeers = r.peers || [];
    renderTraffic();
  } catch (_) {
    /* transient — keep last values */
  } finally {
    trafficBusy = false;
  }
}

function startTraffic() {
  refreshTraffic();
  if (!trafficTimer) trafficTimer = setInterval(refreshTraffic, 5000);
}

function stopTraffic() {
  if (trafficTimer) {
    clearInterval(trafficTimer);
    trafficTimer = null;
  }
}

// --- web panel (advanced limits) -------------------------------------------

async function refreshPanel() {
  try {
    const p = await backend().PanelStatus();
    panelInstalled = !!p.installed;
    $("panel-url").textContent = p.url;
    $("panel-absent").classList.toggle("hidden", p.installed);
    $("panel-present").classList.toggle("hidden", !p.installed);
  } catch (err) {
    panelInstalled = false;
    toast(t("e_check_panel") + errMsg(err), "err");
  }
}

function validPanelPassword(p) {
  return p.length >= 6 && /[a-z]/.test(p) && /[A-Z]/.test(p) && /[0-9]/.test(p) && /[^a-zA-Z0-9]/.test(p);
}

async function installPanel() {
  const pass = $("panel-pass").value;
  if (!validPanelPassword(pass)) {
    toast(t("err_weak_pw"), "err");
    return;
  }
  openLog(t("log_panel"));
  busy(true, t("busy_installing_panel"));
  try {
    await backend().InstallPanel(pass);
    $("panel-pass").value = "";
    await refreshPanel();
    toast(t("toast_panel_installed"), "ok");
  } catch (err) {
    toast(t("e_install_panel") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function removePanel() {
  const ok = await confirmDialog(t("confirm_remove_panel"), t("delete_panel"));
  if (!ok) return;
  busy(true, t("busy_removing_panel"));
  try {
    await backend().RemovePanel();
    await refreshPanel();
    toast(t("toast_panel_removed"), "ok");
  } catch (err) {
    toast(t("e_remove_panel") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function openPanel() {
  try {
    await backend().OpenPanel();
  } catch (err) {
    toast(t("e_open_browser") + errMsg(err), "err");
  }
}

// --- telegram bot ----------------------------------------------------------

const BOT_URLS = {
  botfather: "https://t.me/BotFather",
  userinfo: "https://t.me/userinfobot",
};

async function refreshBot() {
  try {
    const b = await backend().BotStatus();
    $("bot-absent").classList.toggle("hidden", b.installed);
    $("bot-present").classList.toggle("hidden", !b.installed);
  } catch (err) {
    toast(t("e_check_bot") + errMsg(err), "err");
  }
}

async function installBot() {
  const token = $("bot-token").value.trim();
  const admins = $("bot-admins").value.trim();
  const pass = $("bot-pass").value;
  if (!token) {
    toast(t("e_bot_token"), "err");
    return;
  }
  if (!admins) {
    toast(t("e_bot_admins"), "err");
    return;
  }
  if (!validPanelPassword(pass)) {
    toast(t("err_weak_pw"), "err");
    return;
  }
  openLog(t("log_bot"));
  busy(true, t("busy_installing_bot"));
  try {
    await backend().InstallBot(token, admins, pass);
    $("bot-token").value = "";
    $("bot-pass").value = "";
    await refreshBot();
    toast(t("toast_bot_installed"), "ok");
  } catch (err) {
    toast(t("e_install_bot") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function removeBot() {
  const ok = await confirmDialog(t("confirm_remove_bot"), t("delete_bot"));
  if (!ok) return;
  busy(true, t("busy_removing_bot"));
  try {
    await backend().RemoveBot();
    await refreshBot();
    toast(t("toast_bot_removed"), "ok");
  } catch (err) {
    toast(t("e_remove_bot") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

// --- server terminal -------------------------------------------------------

// TERM_CMDS are curated, mostly read-only diagnostics for an AmneziaWG server.
// Clicking one runs it; the user can also type any command.
const TERM_CMDS = [
  "awg show",
  "systemctl status awg-quick@awg0 --no-pager",
  "systemctl status awg-panel --no-pager",
  "journalctl -u awg-quick@awg0 -n 50 --no-pager",
  "ip -br addr",
  "ss -tulpn | grep -E 'awg|8443' || true",
  "uptime",
  "df -h /",
  "free -h",
];

let termRunning = false;

// termWrite appends text to the terminal output and keeps it scrolled to the end.
function termWrite(text) {
  const out = $("term-output");
  out.textContent += text;
  out.scrollTop = out.scrollHeight;
}

// runTerminal executes a command on the server and echoes it + its output, like a
// shell session. Errors (transport failures) are shown inline, not as a toast.
async function runTerminal(cmd) {
  cmd = (cmd || "").trim();
  if (!cmd || termRunning) return;
  termRunning = true;
  termWrite("$ " + cmd + "\n");
  $("term-cmd").value = "";
  try {
    const out = await backend().RunCommand(cmd);
    termWrite(out && out.length ? (out.endsWith("\n") ? out : out + "\n") : "");
  } catch (err) {
    termWrite("⚠ " + errMsg(err) + "\n");
  } finally {
    termWrite("\n");
    termRunning = false;
    $("term-cmd").focus();
  }
}

// renderTermCmds builds the clickable suggested-command chips.
function renderTermCmds() {
  const box = $("term-cmds");
  box.innerHTML = "";
  TERM_CMDS.forEach((cmd) => {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "term-chip mono";
    b.textContent = cmd;
    b.addEventListener("click", () => runTerminal(cmd));
    box.appendChild(b);
  });
}

function initTerminal() {
  renderTermCmds();
  $("term-form").addEventListener("submit", (e) => { e.preventDefault(); runTerminal($("term-cmd").value); });
  $("term-clear").addEventListener("click", () => { $("term-output").textContent = ""; $("term-cmd").focus(); });
}

// --- settings --------------------------------------------------------------

// updateURLs holds the latest CheckUpdate() links for the action buttons.
const updateURLs = { download: "", release: "" };

// checkUpdate asks the backend for the latest release and updates the banner.
async function checkUpdate() {
  const status = $("update-status");
  const dl = $("btn-update-download");
  const notes = $("btn-update-notes");
  dl.classList.add("hidden");
  notes.classList.add("hidden");
  $("update-card").classList.remove("update-on");
  status.textContent = t("update_checking");
  try {
    const u = await backend().CheckUpdate();
    updateURLs.download = u.downloadUrl || "";
    updateURLs.release = u.releaseUrl || "";
    if (u.available) {
      status.textContent = t("update_found", { latest: u.latest, current: u.current });
      dl.classList.remove("hidden");
      notes.classList.remove("hidden");
      $("update-card").classList.add("update-on");
    } else {
      status.textContent = t("update_uptodate", { current: u.current });
    }
  } catch (_) {
    status.textContent = t("update_failed");
  }
}

// loadSettings populates the settings tab: the update banner, the server-info
// card, the rename field, and toggles the panel-password section.
async function loadSettings() {
  checkUpdate();
  $("rename-input").value = ($("srv-label").value || "").trim();
  try {
    const info = await backend().ServerInfo();
    $("info-host").textContent = info.host || "—";
    $("info-port").textContent = info.port || "—";
    $("info-version").textContent = info.version || "—";
    $("info-uptime").textContent = info.uptime || "—";
    $("info-clients").textContent = info.clients != null ? String(info.clients) : "—";
    $("info-panel").textContent = info.panelUrl || "—";
  } catch (err) {
    toast(t("e_server_info") + errMsg(err), "err");
  }
  try {
    const p = await backend().PanelStatus();
    $("settings-panel-pw").classList.toggle("hidden", !p.installed);
    $("settings-panel-absent").classList.toggle("hidden", p.installed);
    $("danger-panel-row").classList.toggle("hidden", !p.installed);
  } catch (_) {
    /* leave defaults if the check fails */
  }
}

async function renameServer() {
  const label = $("rename-input").value.trim();
  busy(true, t("busy_saving"));
  try {
    await backend().RenameServer(label);
    await loadProfiles();
    toast(t("toast_renamed"), "ok");
  } catch (err) {
    toast(t("e_rename") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function changePanelPassword() {
  const pass = $("newpanel-pass").value;
  if (!validPanelPassword(pass)) {
    toast(t("err_weak_pw"), "err");
    return;
  }
  openLog(t("log_change_pass"));
  busy(true, t("busy_changing_pass"));
  try {
    await backend().ChangePanelPassword(pass);
    $("newpanel-pass").value = "";
    toast(t("toast_pass_changed"), "ok");
  } catch (err) {
    toast(t("e_change_pass") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function saveSecondaryEndpoint() {
  const endpoint = $("settings-secondary-endpoint").value.trim();
  busy(true, t("busy_saving"));
  try {
    await backend().SetSecondaryEndpoint(endpoint);
    toast(t("toast_saved", { path: endpoint || t("profile_primary") }), "ok");
  } catch (err) {
    toast(t("e_secondary_endpoint") + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

// --- client result (QR + conf) ---------------------------------------------

function showResult(res) {
  lastResult = res;
  $("result-name").textContent = res.name;
  $("result-endpoints").classList.toggle("hidden", !res.confAlt);
  showResultProfile("primary");
  show($("result"));
  $("result").scrollIntoView({ behavior: "smooth", block: "start" });
}

function showResultProfile(profile) {
  if (!lastResult) return;
  const secondary = profile === "secondary" && lastResult.confAlt;
  $("result-qr").src = secondary ? (lastResult.qrAlt || "") : (lastResult.qr || "");
  $("result-conf").textContent = secondary ? lastResult.confAlt : (lastResult.conf || "");
  $("result-primary").classList.toggle("primary", !secondary);
  $("result-secondary").classList.toggle("primary", !!secondary);
  lastResult.activeProfile = secondary ? "secondary" : "primary";
}

async function downloadConf() {
  if (!lastResult) return;
  try {
    const secondary = lastResult.activeProfile === "secondary";
    const conf = secondary ? lastResult.confAlt : lastResult.conf;
    const name = secondary ? lastResult.name + "-secondary" : lastResult.name;
    const path = await backend().SaveConfig(name, conf);
    if (path) toast(t("toast_saved", { path }), "ok");
  } catch (err) {
    toast(t("e_save") + errMsg(err), "err");
  }
}

// --- wire up ---------------------------------------------------------------

window.addEventListener("DOMContentLoaded", () => {
  applyI18n();
  backend().SetLang(LANG).catch(() => {});
  initAuthTabs();
  initTabs();
  initTrafficSort();
  initTerminal();
  prefill();

  document.querySelectorAll(".langswitch a").forEach((a) => {
    a.addEventListener("click", (e) => { e.preventDefault(); setLang(a.dataset.lang); });
  });

  $("btn-connect").addEventListener("click", connect);
  $("password").addEventListener("keydown", (e) => { if (e.key === "Enter") connect(); });
  $("btn-install").addEventListener("click", install);
  $("add-form").addEventListener("submit", addClient);
  $("btn-uninstall").addEventListener("click", uninstall);
  $("btn-install-panel").addEventListener("click", installPanel);
  $("btn-open-panel").addEventListener("click", openPanel);
  $("btn-remove-panel").addEventListener("click", removePanel);
  $("btn-rename-server").addEventListener("click", renameServer);
  $("btn-change-pass").addEventListener("click", changePanelPassword);
  $("btn-save-secondary-endpoint").addEventListener("click", saveSecondaryEndpoint);
  $("btn-update-check").addEventListener("click", checkUpdate);
  $("btn-update-download").addEventListener("click", () => { if (updateURLs.download) openExternal(updateURLs.download); });
  $("btn-update-notes").addEventListener("click", () => { if (updateURLs.release) openExternal(updateURLs.release); });
  $("btn-install-bot").addEventListener("click", installBot);
  $("btn-remove-bot").addEventListener("click", removeBot);
  // Delegated so the links survive applyI18n() re-rendering them on language switch.
  document.addEventListener("click", (e) => {
    const a = e.target.closest("#link-botfather, #link-userinfo");
    if (!a) return;
    e.preventDefault();
    openExternal(a.id === "link-botfather" ? BOT_URLS.botfather : BOT_URLS.userinfo);
  });
  $("lim-save").addEventListener("click", saveLimits);
  const closeLimits = () => { hide($("limits-modal")); limitsTarget = null; };
  $("lim-cancel").addEventListener("click", closeLimits);
  $("limits-close").addEventListener("click", closeLimits);
  $("limits-modal").addEventListener("click", (e) => { if (e.target === $("limits-modal")) closeLimits(); });
  $("log-close").addEventListener("click", (e) => { e.preventDefault(); e.stopPropagation(); closeLog(); });
  $("result-close").addEventListener("click", () => hide($("result")));
  $("result-download").addEventListener("click", downloadConf);
  $("result-primary").addEventListener("click", () => showResultProfile("primary"));
  $("result-secondary").addEventListener("click", () => showResultProfile("secondary"));
  ["ios", "android", "macos", "windows"].forEach((os) => {
    $("link-" + os).addEventListener("click", (e) => { e.preventDefault(); openExternal(APP_URLS[os]); });
  });
  $("rent-vdsina").addEventListener("click", (e) => { e.preventDefault(); openExternal(RENT_URLS.vdsina); });
  $("rent-hshp").addEventListener("click", (e) => { e.preventDefault(); openExternal(RENT_URLS.hshp); });
  $("btn-switch").addEventListener("click", switchServer);
  $("prompt-cancel").addEventListener("click", () => closePrompt(null));

  if (window.runtime) {
    window.runtime.EventsOn("install:log", appendLog);
    window.runtime.EventsOn("client:log", appendLog);
    window.runtime.EventsOn("panel:log", appendLog);
    window.runtime.EventsOn("bot:log", appendLog);
  }
});
