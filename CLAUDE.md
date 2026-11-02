# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 專案定位

**scan-origin**：依檔名把牙齒掃描 dataset 的 case 分成 **IOS（口內掃描）** 與 **Die（桌面掃描，石膏模切 Die）**，並統計數量。Go 後端（只用標準庫）+ 原生 HTML/JS/CSS 前端（`go:embed` 打包成單一執行檔）。在 dataset 電腦的 dataset 資料夾下執行，開著後端；用另一台電腦（通常是筆電）的瀏覽器連進來看與操作。平台：Windows x64（筆電開發）、Linux x64（ubuntu，dataset 所在）、macOS arm64。發行到 npm（組織 `@cadcam`），在 ubuntu 下載執行。

需求文件在 `doc/`：`261030a_專案目的.md`（目的與使用方式）、`261030b_專案初始prompt.md`（開發慣例來源，參考專案 sermon-proof 在 `D:\coding\信望愛\人工校正語音辨識\`，唯讀）、`261030c_程式架構與細節.md`（資料夾結構與判斷準則）。

**這個工具的本質就是搬動使用者的 dataset 資料夾**（與 sermon-proof 的「原始資料唯讀」不同）。測試一律在 `t.TempDir()` 建立的假資料上跑，**不要對真實 dataset 執行 Apply**。

## 不可違反的原則

- **判斷準則**：case 資料夾（展平後）內存在檔名符合 `…-Tooth_{FDI}.stl`（FDI 兩位數，第一位 1–8、第二位 1–8，不分大小寫）→ Die，否則 IOS。`…-45-crown_cad.stl` 這類雖有 FDI 但不以 `Tooth_XX.stl` 結尾，不算。regex 只在 `internal/dataset/classify.go`，改它要同步看測試。
- **永不覆蓋**：目的地已存在同名 case 就略過並回報，不合併、不刪除；展平遇到名稱衝突就還原並略過。只用 `os.Rename`，不複製、不刪內容；只會 `os.Remove` 空資料夾。
- **先預覽再執行**：UI 一律先 `POST /api/plan`（只讀）給使用者看，再 `POST /api/apply`。Apply 自己重新計算計畫，不信任客戶端。
- **紀錄**：每次 Apply 寫 `<dataset>/.scan-origin/moves-<時間>.json`（所有 mkdir / flatten / move / rename / rmdir），要手動還原時用。
- **安全**：伺服器預設綁 `0.0.0.0` 讓別台電腦連線，所以首頁與 `/api/*` 都要隨機 token（首頁用 `?t=`，API 用 `X-Token`）；非 GET 只收 JSON；`Origin` 必須與 `Host` 一致。啟動時印出含 token 的網址。
- 資料夾權限：新建／碰到的 `IOS`、`Die`、批次資料夾設 777（`chmod777`；Windows 上無實質作用）。
- 前端一律用 `textContent`，case 名稱來自他人上傳，不可用 `innerHTML`。
- **只改使用者要求的檔案**；修改前先看目標檔，不覆蓋未備份內容。
- **不引入第三方套件**（Go 與前端皆然），要加先問。不用 npm 管前端依賴。
- 改動核心邏輯後必須跑 `go vet ./... && go test ./...`。

## 技術備忘

- 資料夾結構：`{dataset}/IOS/{yyMMdd_Count}/{case}`、`{dataset}/Die/{yyMMdd_Count}/{case}`；待處理的上傳是 dataset 根目錄下名稱符合 `^\d{6}_\d+$` 的資料夾。批次資料夾名稱的 Count 是 case 數，Apply 會讓它與實際內容一致（同日期已存在就合併並改名）。
- `{case}/{case}/{files}`（可多層）會先展平成 `{case}/{files}`；分類看展平後的檔案。
- 建置：`go build -o scan-origin .`；測試：`go test ./...`；執行：`./scan-origin [-host ADDR] [-port N] [-no-open] [-here] [dataset資料夾]`（預設埠 8765，被占用且未指定 `-port` 時改用隨機埠）。
- **dataset 資料夾的決定**（`lastdir.go`）：命令列資料夾 > `-here` > 上次記住的 > 目前資料夾。記錄在 `os.UserConfigDir()/scan-origin/last.json`（`{dir, updated}`，先寫 `.tmp` 再 rename），只在伺服器成功監聽後寫入；`-tree` 讀但不寫。用上次資料夾且與目前不同時於 stderr 提醒；上次資料夾不存在則退回 cwd 並警告；`-here` 與資料夾參數並用 → exit 2。因為工具會搬動資料夾，這個提醒與首頁顯示的 dataset 路徑不可拿掉。測試不要動真實的設定資料夾（`chooseDir`／`loadLastDir`／`saveLastDir` 都吃參數，測試用 `t.TempDir()`）。
- **牙位統計**（`internal/dataset/teeth.go`，`GET /api/teeth`，純讀取）：只算已分類的 case；IOS 取 `…-XX-crown_cad.stl`（`_transform` 不算）、Die 取 `…-Tooth_XX.stl`（Die 內的 crown_cad 不看），regex 與 `FDIs()` 在 `classify.go`。一個 case 每個不同 FDI 算一次。因為每個 case 要列一次檔案，所以與便宜的 `/api/state` 分開，前端在 state 之後才請求。
- **匯出 Excel**（`GET /api/export.xlsx`，純讀取）：`internal/dataset/export.go` 組四張表（摘要／各上傳日期／各牙位／Case 清單），`internal/xlsx` 是標準庫（`archive/zip`）寫的最小 xlsx 寫入器（inlineStr、粗體首列、凍結首列、估算欄寬）。下載要帶 `X-Token`，所以前端用 fetch 取 blob 再點暫時連結。`ScanCases` 逐 case 列檔，`ScanTeeth`／匯出都由它而來。xlsx 結構改動後請用真的 Excel 開一次確認（沒有跳修復提示）。
- `-tree`：一次性指令（不啟動伺服器），列出資料夾結構（只有名稱）並用 OSC 52 送到本機終端機的剪貼簿，供 ssh 時貼給 AI；`-depth`／`-sample`／`-dirs-only`／`-o`／`-no-copy` 只能搭配它。純讀取。實作在 `internal/dataset/tree.go` 與 `osc52.go`。
- 發行：`go run ./tools/release`（壓縮檔 + SHA256SUMS）、`go run ./tools/npmpack`（`@cadcam/scan-origin`，optionalDependencies 結構）；版本在 `internal/version`。工具不上傳，`npm publish` 由使用者手動執行。tools 只用標準庫。release 需要至少一個 commit，且追蹤檔無未 commit 的改動。
- 目錄：`internal/dataset`（分類／掃描統計／預覽計畫／Apply）、`internal/server`（HTTP API + token）、`internal/version`、`web/`（前端，embed）、`tools/`、`doc/`（需求文件）。

## Git

- 已 `git init`。**只有使用者要求時才 commit / push**。不要 stage 編譯出的執行檔、`.DS_Store`、`.claude/`。
- Commit 訊息用繁體中文簡短描述。

## ai-log（給未來 AI 的開發紀錄，必須遵守）

- 位置：`ai-log/`。規則與範本見 `ai-log/README.md`，索引見 `ai-log/index.md`。（作法沿用使用者的 sermon-proof 專案。）
- **開始非瑣碎任務前**：先讀 `ai-log/index.md`，再只讀相關的 log。
- **完成有意義的工作單元後**：寫一份 `ai-log/<年>/s<季>/<MMDD><序號>_<english_title>.md`（s1=1–3 月、s2=4–6、s3=7–9、s4=10–12；序號 a…z、aa、ab…；日期用系統當天），並更新 `index.md`。
- **英文撰寫**；目標是只靠 log 就能大致重現程式（行為、資料格式、API、演算法、決策與踩雷），不是只列改了哪些檔。
- 舊 log 不改寫；不寫入金鑰／密碼。目前沒有兄弟專案需要 `Cross-project:` 互指（開發慣例來自 sermon-proof，記在第一份 baseline log）。
