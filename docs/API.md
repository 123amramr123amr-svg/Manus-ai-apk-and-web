# مرجع الـ API / API reference

<div dir="rtl">

كل الطلبات تحت `/api`. أضف مفتاح Manus في الترويسة:

</div>

```http
X-Manus-Api-Key: sk-your-manus-key
Content-Type: application/json
```

<div dir="rtl">

إن كان الخادم مُهيَّأ بـ `MANUS_API_KEY` فلا حاجة للترويسة.

</div>

---

## أمثلة سريعة (curl)

```bash
BASE=http://localhost:8080
KEY="sk-your-manus-key"

# 1) إنشاء مهمة
curl -s -X POST "$BASE/api/tasks" \
  -H "X-Manus-Api-Key: $KEY" -H "Content-Type: application/json" \
  -d '{
        "prompt": "لخّص لي آخر أخبار الذكاء الاصطناعي",
        "title": "ملخص الأخبار",
        "agent_profile": "manus-1.6",
        "locale": "ar-EG",
        "interactive": true
      }'

# 2) متابعة الرسائل
curl -s "$BASE/api/tasks/TASK_ID/messages?limit=50" -H "X-Manus-Api-Key: $KEY"

# 3) إرسال رسالة إضافية
curl -s -X POST "$BASE/api/tasks/TASK_ID/messages" \
  -H "X-Manus-Api-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"message":"أضف جدولًا مقارنًا في النهاية"}'

# 4) رفع ملف ثم إرفاقه بمهمة جديدة
curl -s -X POST "$BASE/api/files" -H "X-Manus-Api-Key: $KEY" -F "file=@report.pdf"
# → {"ok":true,"file_id":"file_xxx","filename":"report.pdf","size_bytes":1234}

curl -s -X POST "$BASE/api/tasks" \
  -H "X-Manus-Api-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"prompt":"حلّل هذا الملف","attachments":[{"file_id":"file_xxx"}]}'

# 5) الرصيد
curl -s "$BASE/api/credits" -H "X-Manus-Api-Key: $KEY"
```

---

## JavaScript (المتصفح)

```js
const api = async (path, options = {}) => {
  const response = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'X-Manus-Api-Key': localStorage.getItem('manus.apiKey') || '',
      ...(options.headers || {})
    }
  });
  const data = await response.json();
  if (!response.ok) throw new Error(data.message || response.statusText);
  return data;
};

const task = await api('/api/tasks', {
  method: 'POST',
  body: JSON.stringify({ prompt: 'اكتب لي خطة تسويق', locale: 'ar-EG' })
});

const messages = await api(`/api/tasks/${task.task_id}/messages`);
```

---

## الأخطاء

<div dir="rtl">

كل خطأ يُرجع JSON بالشكل:

</div>

```json
{ "ok": false, "error": "manus_auth_error", "message": "authentication error (status 401): ..." }
```

| رمز الحالة | `error` | المعنى |
| --- | --- | --- |
| 401 | `missing_api_key` | لا يوجد مفتاح في الطلب ولا على الخادم |
| 401 | `manus_auth_error` | المفتاح مرفوض من Manus |
| 400 | `manus_validation_error` | مدخلات غير صحيحة (أو JSON غير مقبول) |
| 4xx/5xx | `manus_api_error` | خطأ من Manus API (يُمرَّر كما هو) |
| 413 | `file_too_large` | الملف يتجاوز `MANUS_MAX_UPLOAD_BYTES` |
| 502 | `manus_error` | خطأ شبكة/نقل بين الخادم وManus |

---

## مستقبل الـ Webhook

```bash
curl -s -X POST "$BASE/api/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Token: $MANUS_WEBHOOK_TOKEN" \
  -d '{"event_type":"task_stopped","task_detail":{"task_id":"t1","stop_reason":"finish"}}'
```

<div dir="rtl">

الرد يتضمن التصنيف: `kind` ∈ (`created`, `completed`, `stopped`, `waiting_for_input`, `other`)
مع `completed` و `needs_input`. اقرأ الأحداث المخزّنة من `GET /api/webhook/events`.

</div>
