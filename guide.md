[1] question

---

دا شغال اي التحسينات الممكن نعملها

…/.config/agswitch ❯ agswitch ibrahim-wael

Stopping Antigravity...

Switching keyring...

Starting Antigravity...

Done: ibrahim-wael

…/.config/agswitch ❯ agswitch ib2rahim15

Stopping Antigravity...

Switching keyring...

Starting Antigravity...

Done: ib2rahim15

…/.config/agswitch ❯ pwd

/home/ibrahim/.config/agswitch

…/.config/agswitch ❯ ls

Permissions Size User Date Modified Name

.rw-r--r-- 503 ibrahim 24 Jul 16:06  ib2rahim15.json

.rw-r--r-- 503 ibrahim 24 Jul 16:02  ibrahim-wael.json

…/.config/agswitch ❯ cat ~/.local/bin/agswitch

#!/usr/bin/env bash

set -e

PROFILE="$1"

if [ -z "$PROFILE" ]; then

    echo "Usage: agswitch <profile>"

    exit 1

fi

BASE="$HOME/.config/agswitch"

if [ ! -f "$BASE/$PROFILE.json" ]; then

    echo "Profile not found: $PROFILE"

    exit 1

fi

echo "Stopping Antigravity..."

pkill -f Antigravity || true

pkill -f language_server || true

sleep 2

echo "Switching keyring..."

TOKEN=$(cat "$BASE/$PROFILE.json")

secret-tool clear \

service gemini \

username antigravity || true

echo "$TOKEN" | secret-tool store \

--label="Password for 'antigravity' on 'gemini'" \

service gemini \

username antigravity

echo "Starting Antigravity..."

nohup /opt/Antigravity/antigravity >/dev/null 2>&1 &

## echo "Done: $PROFILE"

## [1]answer

---

دا شغال اي التحسينات الممكن نعملها

…/.config/agswitch ❯ agswitch ibrahim-wael

Stopping Antigravity...

Switching keyring...

Starting Antigravity...

Done: ibrahim-wael

…/.config/agswitch ❯ agswitch ib2rahim15

Stopping Antigravity...

Switching keyring...

Starting Antigravity...

Done: ib2rahim15

…/.config/agswitch ❯ pwd

/home/ibrahim/.config/agswitch

…/.config/agswitch ❯ ls

Permissions Size User Date Modified Name

.rw-r--r-- 503 ibrahim 24 Jul 16:06  ib2rahim15.json

.rw-r--r-- 503 ibrahim 24 Jul 16:02  ibrahim-wael.json

…/.config/agswitch ❯ cat ~/.local/bin/agswitch

#!/usr/bin/env bash

set -e

PROFILE="$1"

if [ -z "$PROFILE" ]; then

    echo "Usage: agswitch <profile>"

    exit 1

fi

BASE="$HOME/.config/agswitch"

if [ ! -f "$BASE/$PROFILE.json" ]; then

    echo "Profile not found: $PROFILE"

    exit 1

fi

echo "Stopping Antigravity..."

pkill -f Antigravity || true

pkill -f language_server || true

sleep 2

echo "Switching keyring..."

TOKEN=$(cat "$BASE/$PROFILE.json")

secret-tool clear \

service gemini \

username antigravity || true

echo "$TOKEN" | secret-tool store \

--label="Password for 'antigravity' on 'gemini'" \

service gemini \

username antigravity

echo "Starting Antigravity..."

nohup /opt/Antigravity/antigravity >/dev/null 2>&1 &

echo "Done: $PROFILE"

---

## [2]question

السكربت شغال كويس كبداية، لكن فيه شوية تحسينات مهمة جدًا للأمان والاعتمادية.

## أهم مشكلة حاليًا: صلاحيات ملفات التوكن

الملفات ظاهرة عندك بصلاحية `644`:

```text
.rw-r--r--
```

يعني أي مستخدم محلي على الجهاز يقدر يقرأ محتواها. عدّلها فورًا:

```bash
chmod 700 ~/.config/agswitch
chmod 600 ~/.config/agswitch/*.json
chmod 700 ~/.local/bin/agswitch
```

## نقاط التحسين الأساسية

- `pkill -f language_server` واسع جدًا، وممكن يقفل language server تابع لبرنامج تاني.
- `sleep 2` مدة ثابتة؛ ممكن البرنامج يقفل أسرع أو يحتاج وقت أطول.
- مفيش تأكد إن Antigravity اشتغل فعلًا بعد التشغيل.
- مفيش rollback لو تخزين التوكن الجديد فشل.
- ممكن تشغيل عمليتي `agswitch` في نفس الوقت.
- اسم البروفايل غير متحقق منه، فممكن نظريًا استخدام مسار مثل `../../file`.
- كل أخطاء تشغيل Antigravity بتترمي في `/dev/null`، فصعب تعرف سبب المشكلة.
- استخدام `echo` مع محتوى حساس أقل أمانًا ودقة من تمرير الملف مباشرة إلى `secret-tool`.

## نسخة محسنة

```bash
#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'
umask 077

readonly BASE="${XDG_CONFIG_HOME:-$HOME/.config}/agswitch"
readonly STATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/agswitch"

readonly APP="/opt/Antigravity/antigravity"
readonly SERVICE="gemini"
readonly USERNAME="antigravity"
readonly LABEL="Password for 'antigravity' on 'gemini'"

readonly LOG_FILE="$STATE_DIR/antigravity.log"
readonly CURRENT_FILE="$STATE_DIR/current-profile"

die() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat <<EOF
Usage:
  agswitch <profile>
  agswitch --list
  agswitch --status
  agswitch --help
EOF
}

list_profiles() {
    find "$BASE" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        -printf '%f\n' 2>/dev/null |
        sed 's/\.json$//' |
        sort
}

app_is_running() {
    pgrep -f -- "$APP" >/dev/null 2>&1
}

start_app() {
    nohup "$APP" >>"$LOG_FILE" 2>&1 </dev/null &
}

stop_app() {
    local -a pids=()
    local attempt

    mapfile -t pids < <(pgrep -f -- "$APP" || true)

    if ((${#pids[@]} == 0)); then
        return
    fi

    kill -TERM "${pids[@]}" 2>/dev/null || true

    # انتظر حتى 10 ثوانٍ بدل sleep ثابت
    for ((attempt = 0; attempt < 40; attempt++)); do
        if ! app_is_running; then
            return
        fi

        sleep 0.25
    done

    printf 'Antigravity did not stop gracefully; forcing shutdown...\n' >&2

    mapfile -t pids < <(pgrep -f -- "$APP" || true)

    if ((${#pids[@]} > 0)); then
        kill -KILL "${pids[@]}" 2>/dev/null || true
    fi
}

clear_secret() {
    secret-tool clear \
        service "$SERVICE" \
        username "$USERNAME" \
        >/dev/null 2>&1 || true
}

store_secret_from_file() {
    local source_file="$1"

    secret-tool store \
        --label="$LABEL" \
        service "$SERVICE" \
        username "$USERNAME" \
        <"$source_file"
}

mkdir -p "$BASE" "$STATE_DIR"
chmod 700 "$BASE" "$STATE_DIR"

case "${1:-}" in
    --help|-h)
        usage
        exit 0
        ;;

    --list|-l)
        list_profiles
        exit 0
        ;;

    --status|-s)
        if [[ -f "$CURRENT_FILE" ]]; then
            printf 'Current profile: %s\n' "$(<"$CURRENT_FILE")"
        else
            printf 'Current profile: unknown\n'
        fi

        if app_is_running; then
            printf 'Antigravity: running\n'
        else
            printf 'Antigravity: stopped\n'
        fi

        exit 0
        ;;
esac

PROFILE="${1:-}"

[[ -n "$PROFILE" ]] || {
    usage
    exit 1
}

# يمنع path traversal وأسماء الملفات الغريبة
[[ "$PROFILE" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] ||
    die "Invalid profile name: $PROFILE"

readonly PROFILE_FILE="$BASE/$PROFILE.json"

[[ -f "$PROFILE_FILE" ]] ||
    die "Profile not found: $PROFILE"

[[ ! -L "$PROFILE_FILE" ]] ||
    die "Profile files cannot be symbolic links"

[[ -s "$PROFILE_FILE" ]] ||
    die "Profile is empty: $PROFILE"

chmod 600 "$PROFILE_FILE"

for command_name in secret-tool flock pgrep; do
    command -v "$command_name" >/dev/null 2>&1 ||
        die "Required command not found: $command_name"
done

[[ -x "$APP" ]] ||
    die "Antigravity executable not found: $APP"

# تحقق من صحة JSON لو jq موجود
if command -v jq >/dev/null 2>&1; then
    jq -e . "$PROFILE_FILE" >/dev/null ||
        die "Invalid JSON in profile: $PROFILE"
fi

# منع تشغيل نسختين من agswitch في نفس الوقت
exec 9>"$STATE_DIR/agswitch.lock"

flock -n 9 ||
    die "Another profile switch is already running"

old_secret_file="$(mktemp "$STATE_DIR/old-secret.XXXXXX")"
chmod 600 "$old_secret_file"

had_old_secret=0
switch_started=0
switch_committed=0

if secret-tool lookup \
    service "$SERVICE" \
    username "$USERNAME" \
    >"$old_secret_file" 2>/dev/null; then
    had_old_secret=1
fi

cleanup() {
    local status=$?

    trap - EXIT

    if ((status != 0 && switch_started == 1 && switch_committed == 0)); then
        printf 'Restoring previous keyring entry...\n' >&2

        clear_secret

        if ((had_old_secret == 1)); then
            store_secret_from_file "$old_secret_file" >/dev/null 2>&1 || true
        fi

        if ! app_is_running; then
            start_app || true
        fi
    fi

    rm -f "$old_secret_file"
    exit "$status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

printf 'Stopping Antigravity...\n'
stop_app

printf 'Switching keyring...\n'
switch_started=1

clear_secret
store_secret_from_file "$PROFILE_FILE"

printf 'Starting Antigravity...\n'
start_app

sleep 1

if ! app_is_running; then
    die "Antigravity failed to start. Check: $LOG_FILE"
fi

printf '%s\n' "$PROFILE" >"$CURRENT_FILE"
chmod 600 "$CURRENT_FILE"

switch_committed=1

printf 'Done: %s\n' "$PROFILE"
```

بعد استبدال السكربت:

```bash
chmod 700 ~/.local/bin/agswitch
```

الاستخدام:

```bash
agswitch --list
agswitch ibrahim-wael
agswitch --status
```

وأهم فرق في النسخة دي إن أي فشل أثناء التبديل هيحاول يرجّع التوكن القديم ويشغّل Antigravity من جديد، بدل ما يسيب البرنامج مقفول والـ keyring فاضي. كذلك اللوج أصبح محفوظًا هنا:

```bash
~/.local/state/agswitch/antigravity.log
```

---

## [2]answer

---

النسخة الحالية قوية كـ MVP. علشان تبقى أداة متكاملة فعلًا، ركّز على **معرفة الحالة الحقيقية، الأمان، ثم سهولة الاستخدام** بدل إضافة أوامر شكلية فقط.

## أهم 5 تحسينات تالية

### 1. اجعل `current` يكتشف الحساب الحقيقي

حاليًا ملف:

```text
~/.config/agswitch/current
```

يقول آخر profile استخدمه السكربت، لكنه قد يصبح غير صحيح لو:

- Antigravity غيّر الـ credential.
- سجلت دخول يدويًا بحساب آخر.
- استخدمت `secret-tool` خارج السكربت.
- حصل rollback جزئي.

الأفضل تحسب fingerprint للـ credential الحالي وتقارنه بالبروفايلات:

```bash
credential_hash() {
    sha256sum "$1" | awk '{print $1}'
}
```

ثم تكون النتيجة مثلًا:

```text
Current profile: ibrahim-wael
Credential: matched
Antigravity: running
```

أو:

```text
Current profile: unknown
Credential: not matching any saved profile
Antigravity: running
```

ودي أهم إضافة بعد `save`.

---

### 2. خزّن البروفايلات داخل الـ keyring نفسه

حاليًا ملفات البروفايلات تحتوي credentials صريحة على القرص. صلاحية `600` جيدة، لكنها لا تحمي من:

- backup غير مشفر.
- malware يعمل بنفس المستخدم.
- نسخ المجلد بالخطأ.
- أدوات indexing أو sync.

التصميم الأقوى:

```text
System keyring
├── Active Antigravity credential
├── agswitch profile: ibrahim-wael
└── agswitch profile: ib2rahim15

~/.config/agswitch/
└── metadata فقط، بدون tokens
```

مثال حفظ profile في keyring منفصل:

```bash
secret-tool lookup \
    service gemini \
    username antigravity |
secret-tool store \
    --label="agswitch profile: $PROFILE" \
    service agswitch \
    application antigravity \
    profile "$PROFILE"
```

وعند التبديل:

```bash
secret-tool lookup \
    service agswitch \
    application antigravity \
    profile "$PROFILE" |
secret-tool store \
    --label="Password for 'antigravity' on 'gemini'" \
    service gemini \
    target gemini:antigravity \
    username antigravity
```

الأفضل تجعل الـ keyring هو التخزين الافتراضي، وتدعم الملفات كـ backend اختياري فقط:

```bash
agswitch config set backend keyring
agswitch config set backend files
```

---

### 3. احتفظ بحالة البرنامج الأصلية

حاليًا التبديل يشغّل Antigravity في النهاية حتى لو كان مقفولًا قبل العملية. الأفضل:

```bash
APP_WAS_RUNNING=0

if app_is_running; then
    APP_WAS_RUNNING=1
fi
```

ثم بعد التبديل:

```bash
if ((APP_WAS_RUNNING)); then
    start_app
fi
```

وأضف خيارات:

```bash
agswitch use work
agswitch use work --restart
agswitch use work --no-start
```

السلوك المقترح:

- `use`: يحافظ على حالة البرنامج.
- `--restart`: يشغله في النهاية دائمًا.
- `--no-start`: يبدل الحساب ويتركه مغلقًا.

---

### 4. أضف `doctor`

ده الأمر الأهم لتشخيص أي مشكلة:

```bash
agswitch doctor
```

ويفحص:

```text
[OK] secret-tool installed
[OK] Secret Service is available
[OK] Antigravity executable found
[OK] Config directory permissions: 700
[OK] Profile permissions: 600
[OK] Active credential exists
[OK] Current profile matches active credential
[WARN] 2 active matching keyring entries found
[OK] Antigravity is running
```

يفحص تحديدًا:

- وجود `secret-tool`, `pgrep`, `flock`.
- اتصال D-Bus والـ keyring.
- هل الـ keyring مقفول.
- وجود executable.
- الصلاحيات.
- الملفات الفارغة أو التالفة.
- symlinks.
- duplicate keyring entries.
- تطابق الحساب الحالي.
- إمكانية الكتابة في مجلد اللوج.

ويمكن إضافة إصلاح تلقائي:

```bash
agswitch doctor --fix
```

لكن يخلي أي حذف أو تعديل حساس يحتاج تأكيد.

---

### 5. تحقق بعد تخزين الـ credential

التحقق الحالي يتأكد أن البرنامج اشتغل، لكن ليس أن الـ credential المخزّن يطابق البروفايل المطلوب.

بعد `store`:

```bash
secret-tool lookup \
    service gemini \
    username antigravity \
    > "$VERIFY_FILE"

if ! cmp -s "$PROFILE_FILE" "$VERIFY_FILE"; then
    die "Stored credential does not match the selected profile"
fi
```

الأفضل استخدام hash وعدم طباعة المحتوى:

```bash
expected="$(sha256sum "$PROFILE_FILE" | awk '{print $1}')"
actual="$(sha256sum "$VERIFY_FILE" | awk '{print $1}')"

[[ "$expected" == "$actual" ]] ||
    die "Credential verification failed"
```

بكده العملية تصبح transaction حقيقية:

```text
Preflight → Backup → Stop → Store → Verify → Start → Commit
```

## أوامر مفيدة تكمل الأداة

### تحديث profile موجود

بدل الاعتماد على `save --force` فقط:

```bash
agswitch update work
```

يحفظ الـ credential الحالي فوق `work`.

### إعادة تسمية profile

```bash
agswitch rename old-name new-name
```

### نسخ profile

```bash
agswitch clone work work-backup
```

### معرفة تفاصيل غير حساسة

```bash
agswitch info work
```

مثلًا:

```text
Name: work
Created: 2026-07-24 16:02
Updated: 2026-07-24 18:40
Last used: 2026-07-24 19:10
Storage: keyring
Credential format: JSON
```

لا يعرض token أو حتى جزءًا منه.

### مزامنة الحساب الحالي

لو سجلت دخول يدويًا:

```bash
agswitch detect
```

يعرض:

```text
Active credential matches: ibrahim-wael
```

أو:

```text
Active credential is not saved.

Save it with:
  agswitch save new-profile
```

### تبديل للحساب السابق

مفيد جدًا:

```bash
agswitch previous
```

أو اختصار:

```bash
agswitch -
```

مثل `cd -`.

## تحسين تجربة الاستخدام

### Bash/Zsh/Fish completion

بحيث:

```bash
agswitch use <TAB>
```

يعرض أسماء البروفايلات.

أوامر مقترحة:

```bash
agswitch completion bash
agswitch completion zsh
agswitch completion fish
```

الاستخدام:

```bash
source <(agswitch completion bash)
```

### مخرجات مناسبة للسكربتات

أضف:

```bash
agswitch list --json
agswitch current --quiet
agswitch status --json
```

مثال:

```json
{
  "profile": "ibrahim-wael",
  "credential_matches": true,
  "application_running": true
}
```

مع exit codes ثابتة:

```text
0  نجاح
1  خطأ عام
2  استخدام غير صحيح
3  profile غير موجود
4  credential غير موجود
5  فشل تشغيل التطبيق
6  keyring غير متاح
```

### إشعار Desktop اختياري

```bash
notify-send "Antigravity" "Switched to $PROFILE"
```

ويكون قابلًا للتعطيل:

```bash
agswitch config set notifications false
```

## الـ backup الصحيح

لا تعمل export مباشر إلى JSON غير مشفر. استخدم `age` أو `gpg`:

```bash
agswitch export backup.age
agswitch import backup.age
```

تصميم الأمر:

```bash
tar -C "$BASE" -cf - profiles |
age -p > backup.age
```

واستيراد:

```bash
age -d backup.age |
tar -C "$BASE" -xf -
```

لكن لو انتقلت لتخزين البروفايلات داخل keyring، يحتاج `export` استخراجًا مؤقتًا آمنًا ثم تشفيره مباشرة، مع عدم ترك ملفات plaintext.

## تحسينات محددة في السكربت الحالي

### تحميل المتطلبات حسب الأمر

السكربت حاليًا ينفذ:

```bash
require_tools
```

قبل معرفة الأمر. بالتالي حتى:

```bash
agswitch help
agswitch list
agswitch current
```

قد تفشل لو `secret-tool` أو `flock` غير موجود.

الأفضل كل أمر يطلب احتياجاته فقط:

```bash
case "$command_name" in
    save)
        require_command secret-tool
        require_command flock
        ;;
    use)
        require_command secret-tool
        require_command flock
        require_command pgrep
        ;;
esac
```

### كتابة `current` بصورة atomic

بدل:

```bash
printf '%s\n' "$profile" > "$CURRENT_FILE"
```

استخدم:

```bash
write_current() {
    local profile="$1"
    local temp

    temp="$(mktemp "$BASE/.current.XXXXXX")"
    printf '%s\n' "$profile" >"$temp"
    chmod 600 "$temp"
    mv -f -- "$temp" "$CURRENT_FILE"
}
```

حتى لا يصبح الملف فارغًا لو العملية توقفت أثناء الكتابة.

### تدوير اللوج

الملف:

```text
~/.local/state/agswitch/antigravity.log
```

سيكبر باستمرار. أضف حدًا مثل 5 MB:

```bash
rotate_log() {
    local max_size=$((5 * 1024 * 1024))

    if [[ -f "$LOG_FILE" ]] &&
       (( $(stat -c %s "$LOG_FILE") > max_size )); then
        mv -f "$LOG_FILE" "$LOG_FILE.old"
    fi
}
```

أو الأفضل لاحقًا تشغيل التطبيق عن طريق `systemd --user` واستخدام `journalctl`.

## تحويله إلى مشروع حقيقي

هيكل مناسب:

```text
agswitch/
├── bin/
│   └── agswitch
├── completions/
│   ├── agswitch.bash
│   ├── _agswitch
│   └── agswitch.fish
├── tests/
│   ├── save.bats
│   ├── use.bats
│   └── rollback.bats
├── man/
│   └── agswitch.1
├── install.sh
├── uninstall.sh
├── Makefile
├── README.md
├── CHANGELOG.md
└── LICENSE
```

واستخدم:

```bash
shellcheck bin/agswitch
shfmt -w bin/agswitch
bats tests
```

في الاختبارات اعمل fake executables لـ:

```text
secret-tool
pgrep
flock
antigravity
```

وبكده تقدر تختبر rollback والفشل بدون لمس حساباتك الحقيقية.

## ترتيب التنفيذ الذي أنصح به

**الإصدار القادم:**

```text
1. current ديناميكي باستخدام fingerprint
2. doctor
3. الاحتفاظ بحالة التطبيق الأصلية
4. التحقق من الـ credential بعد التخزين
5. atomic current + log rotation
```

**بعده:**

```text
6. keyring-only profile storage
7. previous / rename / update
8. completion
9. encrypted import/export
10. tests + installer + man page
```

أهم قرار معماري: **انقل secrets من ملفات JSON إلى system keyring، وخلي الملفات للـ metadata فقط**. ده التحسين الذي ينقل `agswitch` من سكربت شخصي جيد إلى credential manager CLI فعلي.

---

## [3]question

طب تمام عايزين نعمل دا و نضيف كمان اني عايز اضيفلو واجها tui عشان افتحو و يقفل اول ما يفتح antigravty و كمان في خاجة ان antigravity مش بيقفل لازم اعمل quit من ال tray icon اديني الخطوات الهنبداء بي عشان نحولو لي مشوروع
و ممكن كمان نخلي يعرف quota لي الاشتراك لكل ايميل و تبان في الواجهة زي
github.com/Draculabo/AntigravityManager@Web search

---

## [3]answer

---

تمام. الأفضل هنا **ما نكمّلاش تكبير سكربت Bash**؛ نخليه نسخة legacy شغالة، ونبني `agswitch` كمشروع حقيقي ببرنامج واحد يحتوي على:

- CLI للأوامر السريعة.
- TUI لإدارة الحسابات والـ quota.
- Keyring لتخزين الحسابات.
- Process manager لتشغيل وإغلاق Antigravity.
- Quota provider مستقل يمكن تغييره لو Google غيّرت الـ API.

أنسب اختيار للمشروع هو **Go** مع Bubble Tea للـ TUI، وCobra للأوامر، و`go-keyring` للتعامل مع Secret Service على Linux. Bubble Tea مصمم لتطبيقات terminal كاملة ويعتمد نموذجًا واضحًا للأحداث والحالة، وCobra يوفر subcommands وcompletion وman pages، بينما `go-keyring` يدعم Linux عبر Secret Service وD-Bus. ([GitHub][1])

# الشكل النهائي المطلوب

تشغيل:

```bash
agswitch
```

يعرض واجهة مثل:

```text
┌ Accounts ──────────────────┬ Account details ───────────────────┐
│                            │                                     │
│ ● ibrahim-wael             │ Email: ibrahim@example.com          │
│   ib2rahim15               │ Plan: Google AI Pro                 │
│   work                     │ Status: Ready                       │
│                            │                                     │
│                            │ Gemini Pro      ███████░░░  72%     │
│                            │ Gemini Flash    █████████░  91%     │
│                            │ Claude Sonnet   ███░░░░░░░  31%     │
│                            │                                     │
│                            │ Reset: 3h 41m                        │
├────────────────────────────┴─────────────────────────────────────┤
│ Enter switch  r refresh  s save  d delete  q quit               │
└──────────────────────────────────────────────────────────────────┘
```

عند الضغط على `Enter`:

```text
✓ Checking profile
✓ Closing Antigravity
✓ Backing up current credential
✓ Activating ibrahim-wael
✓ Verifying keyring
✓ Starting Antigravity
✓ Antigravity is running
```

وبمجرد التأكد أن Antigravity اشتغل، الـ TUI يقفل تلقائيًا ويرجعك للترمينال.

نضيف اختيارًا للبقاء:

```bash
agswitch --stay
```

لكن الافتراضي يكون:

```text
switch → launch → verify → exit TUI
```

# موضوع الـ quota

نعم، نقدر نعرض quota لكل حساب.

مشروع AntigravityManager يجلب بيانات الحساب من endpoints داخلية تشمل:

```text
v1internal:loadCodeAssist
v1internal:fetchAvailableModels
v1internal:retrieveUserQuotaSummary
```

ويستخرج منها:

- نسبة المتبقي لكل model.
- وقت الـ reset.
- subscription tier.
- مجموعات quota.
- AI credits في بعض أنواع الاشتراكات. ([GitHub][2])

لكن كلمة `v1internal` مهمة: دي واجهات داخلية وغير موثقة كـ public stable API. لذلك هننفذها كـ provider منفصل:

```go
type QuotaProvider interface {
    Fetch(ctx context.Context, account Account) (QuotaSnapshot, error)
}
```

والتنفيذ الأول:

```text
InternalGoogleQuotaProvider
```

وبالتالي لو Google غيّرت الـ endpoints أو الـ response schema، نغيّر ملف الـ provider فقط بدل ما المشروع كله يتكسر.

كمان هنضيف:

```text
quota status:
- fresh
- cached
- unavailable
- authentication_required
- rate_limited
```

وممنوع نظهر `0%` لمجرد إن الطلب فشل؛ نظهر `Unavailable` حتى لا تكون المعلومة مضللة.

## طريقة جلب quota لكل profile

لكل حساب:

1. نقرأ credential من keyring.
2. نتحقق من صلاحية `access_token`.
3. لو انتهى، نستخدم `refresh_token`.
4. نجلب بيانات البريد والاشتراك.
5. نجلب quotas الخاصة بالموديلات.
6. نخزن النتيجة مؤقتًا بدون تخزين tokens في الـ cache.

شكل النتيجة:

```json
{
  "email": "ibrahim@example.com",
  "subscription_tier": "Google AI Pro",
  "models": {
    "gemini-pro": {
      "remaining": 72,
      "reset_at": "2026-07-24T22:30:00Z"
    }
  },
  "fetched_at": "2026-07-24T19:00:00Z"
}
```

الـ cache يكون هنا:

```text
~/.cache/agswitch/quota/
```

ولا يحتوي أي `access_token` أو `refresh_token`.

# مشكلة إغلاق Antigravity من الـ tray

دي لازم تكون module مستقلة، لأن `pkill -f` مش حل كامل.

حتى AntigravityManager نفسه، في تنفيذه الحالي، يعمل graceful quit على macOS عبر AppleScript، لكن على Linux ينتقل في النهاية إلى قتل العمليات باستخدام `SIGKILL`؛ يعني كوده الحالي لا يحل مشكلة الـ tray graceful quit على Linux. ([GitHub][3])

هنستخدم سلسلة استراتيجيات:

```text
1. Custom quit command
2. D-Bus tray menu quit
3. SIGTERM للـ main process
4. انتظار الإغلاق
5. SIGKILL للـ process group كحل أخير
```

## الاستراتيجية الأولى: أمر مخصص

نسمح في الإعدادات:

```yaml
process:
  executable: /opt/Antigravity/antigravity
  quit_strategy: auto
  quit_command: ""
  graceful_timeout: 8s
  force_kill: true
```

لو عرفنا command محدد يعمل Quit صح عندك:

```yaml
quit_strategy: command
quit_command: /home/ibrahim/.config/agswitch/quit-antigravity
```

الميزة هنا إن الـ core مش مرتبط بـ GNOME أو KDE أو Wayland.

## الاستراتيجية الثانية: D-Bus tray menu

هنحاول اكتشاف إن كان tray icon مسجلًا كـ:

```text
StatusNotifierItem
com.canonical.dbusmenu
```

لو موجود، نقرأ menu items ونشغّل action اسمه:

```text
Quit
Exit
Close Application
```

دي أفضل طريقة لأنها تعادل الضغط على `Quit` من الـ tray بدل قتل العملية.

لكن لازم نعاملها كميزة Linux desktop-specific؛ طريقة الـ tray تختلف بين GNOME وKDE وبيئات Wayland المختلفة.

## الاستراتيجية الثالثة: SIGTERM للـ main PID فقط

بدل:

```bash
pkill -f Antigravity
```

هنحدد العملية الرئيسية من شجرة العمليات:

```text
Antigravity main process
├── renderer
├── GPU process
├── utility process
└── language server
```

ثم نرسل `SIGTERM` للعملية الرئيسية فقط:

```go
mainProcess.Signal(syscall.SIGTERM)
```

وننتظر مثلًا 8 ثوانٍ.

لو البرنامج ما زال شغالًا، نظهر:

```text
Antigravity did not quit gracefully.

[f] Force close
[c] Cancel
```

وفي التشغيل غير التفاعلي:

```bash
agswitch use work --force
```

## أول تشخيص نعمله على جهازك

شغّل الأوامر دي واحتفظ بالنتيجة داخل ملف، بدون أي tokens:

```bash
mkdir -p ~/Projects/agswitch-diagnostics

{
    echo "Session: $XDG_SESSION_TYPE"
    echo "Desktop: $XDG_CURRENT_DESKTOP"
    echo
    echo "Processes:"
    pgrep -a -f '/opt/Antigravity/antigravity' || true
    echo
    echo "Process tree:"
    pid="$(pgrep -o -f '/opt/Antigravity/antigravity' || true)"
    if [[ -n "$pid" ]]; then
        pstree -aps "$pid"
    fi
    echo
    echo "D-Bus services:"
    busctl --user --no-pager 2>/dev/null |
        grep -Ei 'antigravity|statusnotifier|dbusmenu' || true
} > ~/Projects/agswitch-diagnostics/system.txt
```

وبعدها نجرب الفرق بين:

```bash
kill -TERM "$(pgrep -o -f '/opt/Antigravity/antigravity')"
```

وبين Quit من tray، ونراقب أي D-Bus messages أو تغيرات في شجرة العمليات.

لا أنصح بعمل automation يحرك الماوس ويضغط على مكان tray icon؛ ده هش جدًا ويتكسر مع تغيير ترتيب الأيقونات أو الشاشة أو Wayland.

# معمارية المشروع

```text
agswitch/
├── cmd/
│   ├── root.go
│   ├── tui.go
│   ├── use.go
│   ├── save.go
│   ├── list.go
│   ├── delete.go
│   ├── quota.go
│   ├── doctor.go
│   └── config.go
│
├── internal/
│   ├── account/
│   │   ├── account.go
│   │   └── repository.go
│   │
│   ├── credentials/
│   │   ├── credential.go
│   │   ├── parser.go
│   │   ├── refresher.go
│   │   └── fingerprint.go
│   │
│   ├── keyring/
│   │   ├── active_store.go
│   │   ├── profile_store.go
│   │   └── migration.go
│   │
│   ├── switcher/
│   │   ├── service.go
│   │   ├── transaction.go
│   │   └── rollback.go
│   │
│   ├── process/
│   │   ├── manager.go
│   │   ├── detector_linux.go
│   │   ├── quit_signal_linux.go
│   │   ├── quit_dbus_linux.go
│   │   └── launcher_linux.go
│   │
│   ├── quota/
│   │   ├── provider.go
│   │   ├── google_internal.go
│   │   ├── cache.go
│   │   └── models.go
│   │
│   ├── tui/
│   │   ├── model.go
│   │   ├── update.go
│   │   ├── view.go
│   │   ├── accounts.go
│   │   ├── quota.go
│   │   └── keys.go
│   │
│   ├── config/
│   │   └── config.go
│   │
│   └── logging/
│       └── logging.go
│
├── tests/
│   ├── fixtures/
│   └── integration/
│
├── scripts/
│   ├── install.sh
│   └── migrate-bash-profiles.sh
│
├── .github/workflows/
│   ├── test.yml
│   └── release.yml
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE
└── CHANGELOG.md
```

# تقسيم البيانات

## Secrets

تكون داخل system keyring:

```text
service: agswitch
username: profile:ibrahim-wael

service: agswitch
username: profile:ib2rahim15
```

الـ active credential يظل في المكان الذي يستخدمه Antigravity:

```text
service: gemini
username: antigravity
```

`go-keyring` مناسب لهذا لأنه يتعامل مع Secret Service مباشرة على Linux عبر D-Bus. ([GitHub][4])

## Metadata

هنا فقط بيانات غير حساسة:

```text
~/.config/agswitch/config.yaml
~/.config/agswitch/accounts.json
```

مثال:

```json
{
  "id": "ibrahim-wael",
  "email": "ibrahim@example.com",
  "created_at": "2026-07-24T16:00:00Z",
  "last_used_at": "2026-07-24T19:00:00Z",
  "credential_fingerprint": "sha256:...",
  "quota_enabled": true
}
```

## State

```text
~/.local/state/agswitch/
├── current
├── previous
├── agswitch.log
└── switch.lock
```

# مراحل التنفيذ

## المرحلة 0 — تثبيت السلوك الحالي

قبل كتابة Go:

1. نحتفظ بالسكربت الحالي باسم:

```bash
cp ~/.local/bin/agswitch ~/.local/bin/agswitch-legacy
```

2. نكتب اختبارات سلوكية للنتائج الحالية:

```text
save profile
list profiles
switch profile
rollback on failure
detect running process
start application
```

3. نجهز credential fixture وهمي لا يحتوي على توكنات حقيقية.

4. نحدد schema الفعلية للـ credential بدون إرسال القيم:

```bash
secret-tool lookup service gemini username antigravity |
jq 'walk(if type == "string" then "<redacted>" else . end)'
```

ده يعرض أسماء الحقول فقط تقريبًا مع إخفاء القيم.

## المرحلة 1 — إنشاء المشروع والـ CLI الأساسي

```bash
mkdir -p ~/Projects/agswitch
cd ~/Projects/agswitch

git init
go mod init github.com/ibrahim-wael/agswitch

go get github.com/spf13/cobra@latest
go get charm.land/bubbletea/v2@latest
go get github.com/zalando/go-keyring@latest
```

Cobra يوفر لنا تلقائيًا بنية الأوامر، المساعدة، shell completion وman pages. ([GitHub][5])

أول milestone:

```bash
agswitch list
agswitch current
agswitch save account
agswitch use account --no-launch
agswitch delete account
agswitch doctor
```

بدون TUI أو quota في البداية.

## المرحلة 2 — Keyring ومهاجرة الملفات القديمة

نعمل أمر:

```bash
agswitch migrate
```

يعثر على:

```text
~/.config/agswitch/*.json
```

ثم لكل ملف:

1. يتحقق أنه JSON صالح.
2. يخزنه في keyring.
3. يقرأه مرة ثانية للتحقق.
4. يسجل fingerprint.
5. ينقل الملف إلى backup مشفر أو يطلب حذفه.

الملفات لا تُحذف قبل نجاح التحقق.

## المرحلة 3 — Switch transaction

عملية التبديل تكون:

```text
Acquire lock
    ↓
Validate selected profile
    ↓
Detect current active account
    ↓
Backup active credential in memory/keyring
    ↓
Gracefully close Antigravity
    ↓
Write selected credential
    ↓
Read it again and verify fingerprint
    ↓
Start Antigravity
    ↓
Wait for successful startup
    ↓
Commit current/previous state
```

لو أي خطوة فشلت:

```text
Restore previous credential
Restart previous application state
Report exact failure
```

## المرحلة 4 — حل الإغلاق على جهازك

نعمل interface:

```go
type QuitStrategy interface {
    Name() string
    Available(ctx context.Context) bool
    Quit(ctx context.Context, process Process) error
}
```

والترتيب:

```text
CustomCommandStrategy
DBusTrayStrategy
MainProcessSignalStrategy
ProcessGroupForceStrategy
```

في `doctor` نظهر:

```text
[OK] Antigravity main process detected
[OK] Session type: wayland
[OK] Desktop: KDE
[OK] StatusNotifierItem detected
[OK] Tray menu detected
[OK] Quit menu action detected
```

أو:

```text
[WARN] Tray D-Bus menu not available
[OK] SIGTERM strategy available
[OK] Force-kill fallback available
```

## المرحلة 5 — أول TUI

Bubble Tea مناسب لأن الـ TUI عنده مبني على:

```text
Model → Update → View
```

والـ async operations ترجع كأحداث، وبالتالي تشغيل switch أو quota refresh لن يجمّد الواجهة. ([GitHub][1])

الإصدار الأول من الواجهة يحتوي:

- قائمة الحسابات.
- الحساب الحالي.
- حالة Antigravity.
- `Enter` للتبديل والتشغيل.
- `r` لإعادة القراءة.
- `s` لحفظ الحساب الحالي.
- `d` للحذف.
- status log أسفل الشاشة.
- إغلاق تلقائي بعد نجاح launch.

مكوّنات Bubbles توفر list وtable وprogress وspinner وhelp، وهي مناسبة مباشرة لقائمة الحسابات وأشرطة quota. ([GitHub][6])

## المرحلة 6 — البريد والـ quota

نبدأ يدويًا:

```bash
agswitch quota ibrahim-wael
```

الناتج:

```text
Account: ibrahim@example.com
Plan: Google AI Pro

Gemini Pro     72% remaining   resets in 3h 41m
Gemini Flash   91% remaining   resets in 3h 41m
Claude Sonnet  31% remaining   resets in 18h
```

ثم:

```bash
agswitch quota --all
```

بعد التأكد من الثبات، ندمجها داخل الـ TUI.

لا نضيف auto-switch في نفس المرحلة. الأول لازم نتأكد أن quota data صحيحة؛ يوجد بلاغات في AntigravityManager عن حالات تعرض فيها الواجهة `100%` رغم حصول المستخدم على `429`، ما يؤكد ضرورة التفريق بين quota المعروضة وأخطاء rate limiting الفعلية. ([GitHub][7])

## المرحلة 7 — Auto-switch

بعد استقرار quota:

```yaml
auto_switch:
  enabled: false
  threshold: 5
  strategy: highest_remaining
  refresh_interval: 5m
```

السلوك:

```text
Current account < 5%
        ↓
Find healthy profiles
        ↓
Sort by quota/reset time
        ↓
Notify user
        ↓
Switch only when Antigravity is idle
```

لا أنصح بالتبديل التلقائي أثناء وجود request شغال في Antigravity في الإصدار الأول.

## المرحلة 8 — الاختبارات والإصدارات

اختبارات أساسية:

```text
keyring mock tests
credential parser tests
switch rollback tests
process-tree detection tests
quota response fixture tests
TUI state transition tests
```

والـ releases:

```text
agswitch_linux_amd64
agswitch_linux_arm64
```

مع:

```bash
make test
make build
make install
```

# تنبيه متعلق بـ AntigravityManager

نقدر ندرس تصميمه والـ API behavior، لكن ما ننقلش كوده مباشرة. المشروع منشور بترخيص `CC BY-NC-SA 4.0`، ويتضمن شرط الاستخدام غير التجاري وShareAlike للأعمال المشتقة؛ لذلك الأفضل نكتب التنفيذ من الصفر اعتمادًا على السلوك والبروتوكول المطلوب، ونختار ترخيص مشروعنا من البداية بوضوح. ([GitHub][8])

# أول Sprint نبدأ به

نطاق أول نسخة يكون فقط:

```text
[1] Go project skeleton
[2] CLI commands
[3] System keyring profile storage
[4] Migration من ملفات JSON
[5] Process detection
[6] Quit strategy diagnostics
[7] Transactional switching + rollback
[8] Minimal TUI
[9] Exit TUI after Antigravity starts
```

الـ quota تدخل في الـ sprint التالي؛ لأنها تعتمد على API داخلية وOAuth refresh، بينما التبديل والإغلاق لازم يكونا مستقرين الأول. أول خطوة عملية الآن هي حفظ ناتج `system.txt` من أوامر التشخيص، ثم بناء skeleton المشروع والـ keyring interfaces قبل كتابة شكل الواجهة.

[1]: https://github.com/charmbracelet/bubbletea "GitHub - charmbracelet/bubbletea: A powerful little TUI framework  · GitHub"
[2]: https://github.com/Draculabo/AntigravityManager/blob/main/src/modules/cloud-account/services/GoogleAPIService.ts "AntigravityManager/src/modules/cloud-account/services/GoogleAPIService.ts at main · Draculabo/AntigravityManager · GitHub"
[3]: https://github.com/Draculabo/AntigravityManager/blob/main/src/modules/antigravity-runtime/ipc/handler.ts "AntigravityManager/src/modules/antigravity-runtime/ipc/handler.ts at main · Draculabo/AntigravityManager · GitHub"
[4]: https://github.com/zalando/go-keyring "GitHub - zalando/go-keyring: Cross-platform keyring interface for Go · GitHub"
[5]: https://github.com/spf13/cobra "GitHub - spf13/cobra: A Commander for modern Go CLI interactions · GitHub"
[6]: https://github.com/charmbracelet/bubbles "GitHub - charmbracelet/bubbles: TUI components for Bubble Tea  · GitHub"

[7]: https://github.com/Draculabo/AntigravityManager/issues/103?utm_source=chatgpt.com "[Bug] Pro models return 429 \"Resource Exhausted\" despite 100 ..."
[8]: https://github.com/Draculabo/AntigravityManager/blob/main/LICENSE "AntigravityManager/LICENSE at main · Draculabo/AntigravityManager · GitHub"
