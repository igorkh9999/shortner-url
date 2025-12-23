# How to Add k6 to PATH on Windows

## Step 1: Download and Extract k6

1. Download k6 from: https://github.com/grafana/k6/releases/latest
2. Look for `k6-v0.x.x-windows-amd64.zip` (or `k6-v0.x.x-windows-arm64.zip` for ARM)
3. Extract the zip file to a permanent location, for example:
    - `C:\k6`
    - `C:\Program Files\k6`
    - `C:\tools\k6`

## Step 2: Add to PATH

### Method A: Using GUI (Easiest)

1. **Press `Win + X`** and select **"System"**
2. Click **"Advanced system settings"** (on the right side)
3. Click **"Environment Variables"** button at the bottom
4. Under **"User variables"** section, find and select **"Path"**
5. Click **"Edit..."**
6. Click **"New"**
7. Enter the path where you extracted k6 (e.g., `C:\k6`)
8. Click **"OK"** on all dialogs
9. **Close and reopen** your terminal/command prompt
10. Test with: `k6 version`

### Method B: Using PowerShell (As Administrator)

Open PowerShell as Administrator and run:

```powershell
# Replace C:\k6 with your actual k6 installation path
$k6Path = "C:\k6"
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
$newPath = $currentPath + ";$k6Path"
[Environment]::SetEnvironmentVariable("Path", $newPath, "User")
```

Then **close and reopen** your terminal.

### Method C: Using Command Prompt (As Administrator)

Open Command Prompt as Administrator and run:

```cmd
setx PATH "%PATH%;C:\k6"
```

(Replace `C:\k6` with your actual k6 installation path)

Then **close and reopen** your terminal.

## Step 3: Verify Installation

Open a **new** terminal and run:

```cmd
k6 version
```

You should see the k6 version information.

## Troubleshooting

-   **"k6 is not recognized"**: Make sure you closed and reopened your terminal after adding to PATH
-   **"Access denied"**: Run PowerShell/Command Prompt as Administrator
-   **Wrong path**: Double-check the path where you extracted k6

## Alternative: Run Without PATH

If you don't want to add to PATH, you can always run k6 using the full path:

```cmd
C:\k6\k6.exe version
C:\k6\k6.exe run C:\playground\shortner-url\load-test\script.js
```
