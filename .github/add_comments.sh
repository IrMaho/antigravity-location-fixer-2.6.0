#!/bin/bash

# Comments to add to each issue
declare -A comments

# Issue #4 - مشکل شناسایی پروفایل (فارسی)
comments[4]="سلام @3edighi! این bug معروفی است که وقتی session backup می‌گیری، profile ها رو شناسایی نمی‌کنه.

**نسخه جدید** این مسئله رو حل کرده است. بک‌آپ session و شناسایی profile‌ها الآن بدون مشکل کار می‌کنه. بزودی می‌تونی اپدیت رو دریافت کنی. 🚀"

# Issue #5 - Your current account is not eligible for Antigravity (انگلیسی)
comments[5]="Hello @Swench44! Thank you for reporting this issue.

This error is a limitation on Google's side, not our app. However, the **new version coming soon** handles this better and provides a clearer error message.

The new update also improves account compatibility across all operating systems (macOS, Linux, Windows). We're working on better Google account support in the upcoming release."

# Issue #7 - Linux/macOS installation (انگلیسی)
comments[7]="Hi @rzgrozt! 😄 You're absolutely right!

We apologize for that gap. The app was Windows-only. **We're releasing a new version soon** that will be **fully compatible with all operating systems** (macOS, Linux, and Windows). Native installation scripts for Unix-based systems are included. This is one of our top priorities.

Thank you for pushing us to do better!"

# Issue #8 - install.sh missing - Installation on mac not working (انگلیسی)
comments[8]="Hello @youngrichu! You're correct - the install.sh file for macOS was missing.

**The new version coming soon solves this completely.** We're adding native support for macOS and Linux with proper installation scripts. All operating systems will have a proper installation experience. The update will be available shortly."

# Issue #9 - نسخه مک (فارسی)
comments[9]="سلام @holyprot! نسخه macOS الان disabled است چون فقط PowerShell support داشتیم.

اما **نسخه جدید بزودی میاد** که **برای تمام سیستم عامل‌ها** (macOS، Linux، Windows) بهینه شده. نسخه native macOS الآن فعال است و بدون مشکل کار می‌کنه. 🚀

منتظر آپدیت باشید!"

echo "Comments prepared for all issues"
