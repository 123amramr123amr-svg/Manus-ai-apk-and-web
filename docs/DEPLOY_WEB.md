# نشر تطبيق الويب / Deploying the web app

<div dir="rtl">

تطبيق الويب عبارة عن ملف تنفيذي واحد (Go) يحتوي على الواجهة والـ API معًا — لا حاجة إلى Node.js ولا إلى قاعدة بيانات.

</div>

---

## 1) تشغيل محلي

```bash
go run ./cmd/manusweb              # http://localhost:8080
PORT=9000 go run ./cmd/manusweb    # منفذ مختلف
```

<div dir="rtl">

ثم افتح `http://localhost:8080` ← ⚙ ← ألصق مفتاح Manus API ← **اختبار الاتصال**.

</div>

---

## 2) Docker

```bash
docker build -t manus-ai-web .
docker run -d --name manus-web -p 8080:8080 \
  -e MANUS_API_KEY="sk-..." \
  -e MANUS_WEBHOOK_TOKEN="a-long-random-secret" \
  manus-ai-web
```

<div dir="rtl">

أو باستخدام `docker-compose.yml` الموجود في المستودع: `docker compose up -d --build`.

</div>

---

## 3) Render (مجاني)

<div dir="rtl">

1. ادفع المستودع إلى GitHub.
2. في Render: **New → Blueprint** واختر المستودع (سيقرأ `render.yaml`).
3. اترك `MANUS_API_KEY` فارغًا إن أردت أن يُدخل كل مستخدم مفتاحه، أو ضع مفتاحًا لمشاركته.
4. الرابط الناتج (مثل `https://manus-ai-web.onrender.com`) هو ما تضعه في تطبيق الأندرويد.

</div>

## 4) Fly.io

```bash
fly launch --no-deploy --dockerfile Dockerfile
fly secrets set MANUS_API_KEY="sk-..." MANUS_WEBHOOK_TOKEN="secret"
fly deploy
```

## 5) VPS (systemd)

```bash
go build -o /usr/local/bin/manusweb ./cmd/manusweb
```

```ini
# /etc/systemd/system/manusweb.service
[Unit]
Description=Manus AI web app
After=network.target

[Service]
Environment=PORT=8080
Environment=MANUS_API_KEY=sk-...
Environment=MANUS_WEBHOOK_TOKEN=secret
ExecStart=/usr/local/bin/manusweb
Restart=always
User=manus

[Install]
WantedBy=multi-user.target
```

<div dir="rtl">

ضع Nginx أو Caddy أمامه للحصول على HTTPS (مطلوب لخاصية PWA ولمستقبل Webhooks).

</div>

---

## 6) Webhooks (اختياري)

<div dir="rtl">

لتلقي إشعارات Manus داخل التطبيق:

1. انشر الخادم على رابط عام بـ HTTPS.
2. من التطبيق ← تبويب **الأدوات** ← **Webhooks** ← اضغط **استخدم هذا الخادم** ← **تسجيل**.
3. إن ضبطت `MANUS_WEBHOOK_TOKEN` فسيكون عنوان التسجيل:
   `https://your-app.example.com/api/webhook?token=a-long-random-secret`
4. تظهر الأحداث الواردة في تبويب **النشاط**.

> ملاحظة: الأحداث تُحفظ في الذاكرة (آخر 200 حدث افتراضيًا) وتُفقد عند إعادة تشغيل الخادم.

</div>

---

## 7) ملاحظات أمنية

<div dir="rtl">

* مفتاح المستخدم يبقى في `localStorage` داخل متصفحه، والخادم يمرّره فقط إلى Manus API.
* الخادم لا يخزّن المفاتيح ولا يكتبها في السجلّات.
* ترويسات الحماية المفعّلة: `CSP`, `X-Content-Type-Options`, `X-Frame-Options: DENY`, `Referrer-Policy`.
* حد رفع الملفات افتراضيًا 25MB (قابل للتعديل بـ `MANUS_MAX_UPLOAD_MB`).
* إن كنت تنشر التطبيق للعامة، لا تضع مفتاحًا في `MANUS_API_KEY`، وفكّر في إضافة طبقة مصادقة أمام الخادم.

</div>
