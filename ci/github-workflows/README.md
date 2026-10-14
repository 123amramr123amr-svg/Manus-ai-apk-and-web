# GitHub Actions workflows (copy these into `.github/workflows/`)

<div dir="rtl">

هذان الملفان هما نفس ملفات GitHub Actions المذكورة في التوثيق، لكنهما موضوعان هنا لأن
بيئة التوليد هذه لا تملك صلاحية `workflows` لإنشاء ملفات داخل `.github/workflows/` مباشرة.

**كيف تستخدمهما؟** اختر إحدى الطرق:

1. من واجهة GitHub: **Add file → Create new file**، واكتب المسار `.github/workflows/build-apk.yml`
   ثم الصق محتوى `ci/github-workflows/build-apk.yml` وارفعه. كرّر الأمر مع `ci.yml`.
2. من جهازك (بعد استنساخ المستودع):

```bash
mkdir -p .github/workflows
cp ci/github-workflows/*.yml .github/workflows/
git add .github/workflows && git commit -m "ci: add workflows" && git push
```

</div>

* `ci.yml` — يشغّل `gofmt` و `go vet` و `go test -race` وبناء الخادم واختبارًا سريعًا وبناء صورة Docker.
* `build-apk.yml` — يبني الـ APK عبر Gradle، يرفعه كـ Artifact، ويرفقه بصفحة Releases عند دفع وسم (`v*`).

<div dir="rtl">

> إن لم ترغب في استخدام Actions أصلًا، يمكنك بناء الـ APK محليًا كما في [`docs/BUILD_APK.md`](../docs/BUILD_APK.md).

</div>
