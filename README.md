<div dir="rtl">

# 🚀 فیکسر لوکیشن و تحریم آنتی‌گراویتی آیدیا ۲.۶.۰ (Antigravity Location Fixer)
### ابزار جامع رفع تحریم، ارور ۴۰۳، آنلاک دسترسی اینترنال و رفع مشکل لوپ دانلود بک‌اند در Antigravity IDE v2.6.0

[![GitHub Stars](https://img.shields.io/github/stars/IrMaho/antigravity-location-fixer-2.6.0?style=social)](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/IrMaho/antigravity-location-fixer-2.6.0?style=social)](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/network/members)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22%20--%201.25+-00ADD8?logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-informational)](https://github.com/IrMaho/antigravity-location-fixer-2.6.0)

---

</div>

<p align="center">
  <a href="#فارسی">🇮🇷 راهنمای فارسی</a> •
  <a href="#english">🇺🇸 English Guide</a>
</p>

---

<div dir="rtl" id="فارسی">

## 📌 درباره پروژه و چرایی ساخت نسخه ۲.۶.۰

در نسخه‌های قبلی آنتی‌گراویتی (مانند ۲.۵.۵)، پچرهای قدیمی کار می‌کردند؛ اما با انتشار **نسخه ۲.۶.۰ Antigravity IDE**، شرکت گوگل ساختار ادیتور را به طور اساسی تغییر داد که باعث از کار افتادن پچرهای قبلی و ایجاد مشکلات زیر شد:

1. **گیر کردن در لوپ دانلود بی‌پایان (`Installing Antigravity Backend: Downloading 10%...`):**
   - در نسخه ۲.۶.۰ کدهای فایل `extension.js` مینیفای شدند و متغیرهایی مثل `isChannelChanged` به `Pc` تغییر نام دادند. این موضوع باعث می‌شد هربار که شورت‌کات جدیدی باز می‌شد یا بین اکانت‌ها سوئیچ می‌کردید، ادیتور تصور کند بک‌اند نامعتبر است و شروع به دانلود مجدد کند و روی ۱۰٪ فریز شود.
2. **تغییر ساختار داخلی از `app.asar` به پوشه باز (`resources/app/`):**
   - ادیتور دیگر از فایل فشرده `app.asar` استفاده نمی‌کرد و پچرهای قبلی با خطای عدم یافتن فایل کرش می‌کردند.
3. **تغییر نام پروسه‌ها و مسیر بک‌اند:**
   - پروسه ادیتور به `Antigravity IDE.exe` و باینری هسته به `%USERPROFILE%\.gemini\bin\agy.exe` منتقل شد.
4. **خطای مسدودیت جغرافیایی و تحریم (Error 403 Forbidden):**
   - باینری `language_server` و `agy.exe` در زمان لاگین روی اکانت‌های جدید یا سوئیچ اکانت، موقعیت مکانی کاربر را چک کرده و دسترسی را قطع می‌کردند.

**این ریپازیتوری تمام این ۴ مشکل را به طور ۱۰۰٪ قطعی، تست‌شده و پایدار حل کرده است.**

---

## ✨ قابلیت‌های اصلی

- ✅ **پچ هوشمند `extension.js` (رفع قطعی لوپ دانلود):** تزریق گارد جلوگیری از بررسی مجدد نسخه تا دیگر هرگز پیام Downloading Backend نشان داده نشود.
- ✅ **پشتیبانی کامل از نسخه ۲.۶.۰ و پوشه Unpacked:** تشخیص هوشمند فایل‌های باز `resources/app/` بدون نیاز به ابزارهای اکسترکت اسار.
- ✅ **بای‌پس چنددرگاهی تحریم (MultiGate Bytecode Unlock):** باز کردن قفل موقعیت مکانی و لایسنس در باینری‌های `agy.exe` و `language_server.exe`.
- ✅ **فعال‌سازی دسترسی اینترنال ادیتور (`isGoogleInternal -> true`):** پچ فایل `main.js` برای دریافت دسترسی‌های اینترنال توسعه‌دهندگان گوگل.
- ✅ **حفاظت هوشمند از پروسه‌ها (Session Protection):** جلوگیری از بسته‌شدن ناگهانی پنجره‌های ترمینال یا نشست‌های فعال هوش مصنوعی هنگام اجرای پچ.
- ✅ **سیستم عیب‌یابی و پایش سلامت سیستم (`doctor`):** بررسی کامل اتصالات سرور گوگل، Gemini API، وضعیت لایسنس و سلامت شبکه.

---

## 📦 دانلود مستقیم نسخه‌های آماده (Direct Portable Download)

نیازی به نصب یا کامپایل هیچ پیش‌نیازی ندارید! می‌توانید فایل اجرایی سیستم‌عامل خود را مستقیماً دریافت کنید:

| سیستم‌عامل / پلتفرم | نوع فایل | لینک دانلود مستقیم از Release |
| :--- | :--- | :--- |
| 🖥️ **نسخه گرافیکی ویندوز (پیشنهادی)** | بدون نیاز به کد و ترمینال (`.exe`) | [📥 دانلود antigravity-gui.exe](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-gui.exe) |
| 🪟 **آرشیو نسخه گرافیکی ویندوز** | ZIP پرتابل بدون کنسول | [📥 دانلود نسخه گرافیکی (.zip)](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-location-fixer-v2.6.0-windows-gui.zip) |
| 🪟 **نسخه ترمینال ویندوز (CLI)** | فایل اجرایی مستقیم (`.exe`) | [📥 دانلود antigravity.exe](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity.exe) |
| 🪟 **آرشیو ترمینال ویندوز** | ZIP نسخه کنسول | [📥 دانلود نسخه CLI (.zip)](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-location-fixer-v2.6.0-windows-amd64.zip) |
| 🐧 **لینوکس (Linux x86_64)** | باینری لینوکس 64 بیتی | [📥 دانلود نسخه لینوکس 64](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-location-fixer-linux-amd64) |
| 🐧 **لینوکس (Linux ARM64)** | باینری لینوکس ARM | [📥 دانلود نسخه لینوکس ARM64](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-location-fixer-linux-arm64) |
| 🍎 **مک‌او‌اس (macOS Apple Silicon)** | باینری تراشه‌های M1 تا M4 | [📥 دانلود نسخه مک اپل سیلیکون](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-location-fixer-darwin-arm64) |
| 🍎 **مک‌او‌اس (macOS Intel)** | باینری اینتل مک | [📥 دانلود نسخه مک اینتل](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-location-fixer-darwin-amd64) |

---

## 🎯 آموزش ۰ تا ۱۰۰ پچ کردن و ورود به اکانت (بدون نیاز به یک خط کد!)

اگر با ترمینال یا دستورات کامپیوتری آشنایی ندارید، این راهنما دقیقاً برای شماست:

### گام ۱: دانلود برنامه گرافیکی
- روی لینک [📥 دانلود مستقیم antigravity-gui.exe](https://github.com/IrMaho/antigravity-location-fixer-2.6.0/releases/download/v2.6.0/antigravity-gui.exe) کلیک کنید.

### گام ۲: بستن ادیتور
- اگر برنامه Antigravity IDE در حال اجراست، آن را ببندید تا فایل‌های برنامه آزاد باشند.

### گام ۳: اجرای برنامه گرافیکی با یک دابل‌کلیک
- روی فایل `antigravity-gui.exe` دو بار کلیک کنید.
  > 💡 **نکته در مورد پیام آبی‌رنگ ویندوز (SmartScreen):**  
  > اگر پنجره آبی‌رنگ **Windows protected your PC** باز شد، این یک هشدار طبیعی ویندوز برای برنامه‌های تازه کامپایل‌شده متن‌باز است. کافیست روی گزینه آبی‌رنگ **`More info`** کلیک کنید و سپس دکمه **`Run anyway`** را بزنید تا برنامه باز شود.
- یک پنجره گرافیکی و کلاسیک زیبا (بدون هیچ صفحه سیاه CMD) باز می‌شود.
- برنامه به صورت خودکار مسیر نصب Antigravity IDE را روی ویندوز شما پیدا می‌کند.

### گام ۴: زدن دکمه Auto-Fix
- کافیست روی دکمه اول: **`① ⚡ Auto-Fix`** کلیک کنید.
- چند ثانیه بعد نوار وضعیت سبز شده و پیام زیر را نمایش می‌دهد:
  `✔ All 4 targets patched & Zero-Telemetry enabled!`
- برنامه به طور خودکار:
  1. مشکل گیر کردن روی دانلود ۱۰٪ (`Downloading Backend`) را برطرف می‌کند.
  2. قفل تحریم و چک لوکیشن (ارور ۴۰۳ گوگل) را باز می‌کند.
  3. دسترسی‌های اینترنال توسعه‌دهندگان گوگل را فعال می‌کند.
  4. شورت‌کات دسکتاپ بهینه‌شده را ایجاد می‌کند.

### گام ۵: ورود به اکانت (Login)
1. فیلترشکن یا پروکسی سیستمتان (مثل V2RayN یا Clash) را روشن کنید.
2. برنامه Antigravity IDE را باز کنید.
3. روی دکمه آبی‌رنگ **Continue with Google** کلیک کنید.
4. صفحه ورود در یک تب جدید در همان مرورگر گوگل کروم باز می‌شود که اکانت‌هایتان در آن لاگین هستند.
5. روی اکانت خود کلیک کنید تا وارد شوید. ادیتور فوراً متصل شده و تمامی مدل‌های هوش مصنوعی فعال می‌شوند!

---

## 💻 راهنمای متنی برای کاربران ترمینال (اختیاری)

اگر ترجیح می‌دهید از خط فرمان استفاده کنید:

```powershell
# بررسی سلامت اتصالات و پچ‌ها:
.\antigravity.exe doctor

# اجرای پچ خودکار کامل با یک دستور:
.\antigravity.exe autofix
```

---

## 🛠️ ساخت از سورس (Build from Source)

اگر می‌خواهید خودتان پروژه را از صفر کامپایل کنید:

```bash
# نیازمند Go نسخه 1.22 یا بالاتر
git clone https://github.com/IrMaho/antigravity-location-fixer-2.6.0.git
cd antigravity-location-fixer-2.6.0

# کامپایل فایل اجرایی
go build -o antigravity.exe ./cmd/antigravity

# اجرای تست‌ها
go test -count=1 ./...
```

---

## 📋 جدول وضعیت خروجی Doctor پس از پچ

پس از اجرای موفق پچ، خروجی دستور `.\antigravity.exe doctor` باید به این شکل باشد:

```text
╭────────────────────────────────────────────────────────────────────────────╮
│  🩺 Antigravity Cleaner — Health & Connectivity Audit                      │
│ ────────────────────────────────────────────────────────────────────────   │
│  OS Environment          READY   Microsoft Windows NT [amd64]              │
│  Antigravity Build       READY   Antigravity IDE (v2.6.0)                  │
│  Core Engine             READY   Unlocked (Patched)                        │
│  IDE Interface           READY   Unlocked (Internal Mode)                  │
│  Active Process          READY   Idle / Closed                             │
│  Recent Logs             READY   Clean (No fatal crashes in logs)          │
╰────────────────────────────────────────────────────────────────────────────╯
   ONLINE  System is optimized. You can launch Antigravity with Smart Proxy!
```

</div>

---

<div id="english">

## 🇺🇸 English Documentation

### Overview
**Antigravity Location Fixer & Patcher (v2.6.0)** is an autonomous multi-target bytecode patcher and sanction bypass tool specifically engineered for **Google Antigravity IDE v2.6.0**.

### What changed in v2.6.0?
- **Unpacked App Directory:** Replaced legacy `app.asar` with an unpacked `resources/app/` structure.
- **Minified VS Code Extension Symbols:** Minified symbol signatures in `vscode/extension.js` (`isChannelChanged` became `Pc`, channel logging changed), which previously broke patch regexes and caused the editor to enter an infinite loop downloading `win32-x64` backend binaries at 10%.
- **Binary Re-location:** Backend relocated to `%USERPROFILE%\.gemini\bin\agy.exe`.
- **403 Region Lock:** Strict client-side eligibility validation on core binaries.

### Key Capabilities
- **Backend Download Guard:** Eliminates the annoying `Installing Antigravity Backend: Downloading (10%)...` hang on isolated account windows.
- **MultiGate Bytecode Unlock:** Neutralizes regional restrictions (Error 403 Forbidden) directly inside `agy.exe` and `language_server.exe`.
- **Internal Privilege Mode:** Patches `main.js` (`isGoogleInternal -> true`) for access to full developer tiers.
- **Integrated Session Safety:** Protects parent terminal and AI chat processes from being killed during patch application.

### Quick Start
```powershell
# 1. Audit system health
.\antigravity.exe doctor

# 2. 1-Click Auto-Fix
.\antigravity.exe autofix
```

### Build from Source
```bash
go build -o antigravity.exe ./cmd/antigravity
```

---

## 📄 License
This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Developed with ❤️ by [IrMaho](https://github.com/IrMaho).

</div>
