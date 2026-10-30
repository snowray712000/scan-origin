# scan-origin

依檔名把牙齒掃描 dataset 的 case 分成 **IOS（口內掃描）** 與 **Die（桌面掃描，石膏模切 Die）**，並統計數量。Go 標準庫 + 原生 HTML/JS/CSS，單一執行檔。

## 安裝

發行檔提供 Windows x64、macOS Apple Silicon、Linux x64。

```bash
npm i -g @cadcam/scan-origin     # 需要 Node.js 16+，npm 會自動挑對應系統的執行檔
```

或到 GitHub Release 下載壓縮檔（`scan-origin_<版本>_<windows|darwin|linux>_<amd64|arm64>`，附 `SHA256SUMS.txt`），解壓後直接執行。目前沒有數位簽章與 Apple 公證，第一次執行 macOS（Gatekeeper）或 Windows（SmartScreen）可能跳警告。

## 使用

在 dataset 電腦的 dataset 資料夾下執行：

```bash
scan-origin                      # 以目前資料夾為 dataset
scan-origin /path/to/dataset     # 或指定資料夾
```

啟動後會印出含 token 的網址，用另一台電腦的瀏覽器開啟（例如 `http://192.168.1.20:8765/?t=…`）：

- **Dataset 目前狀態**：已分類總計、IOS 數、Die 數、待處理數，以及每個上傳日期的 IOS／Die 數。
- **整理待處理的上傳**：先按「預覽整理結果」看每個 case 會被判成 IOS 或 Die，確認後才「執行」。

選項：`-host`（預設 `0.0.0.0`；只給本機用請設 `127.0.0.1`）、`-port`（預設 8765，被占用且未指定時改用隨機埠）、`-no-open`（不在本機開瀏覽器）、`-version`。結束用 Ctrl+C。

> 預設監聽所有網卡，Windows 第一次執行可能跳防火牆提示。首頁與 API 都需要啟動時產生的隨機 token，**知道含 token 的網址就能整理這個資料夾**，請勿外流；每次啟動 token 都會改變。沒有 TLS，請只在可信任的內網使用。

## 資料夾結構與判斷準則

```
{dataset}/
  IOS/261022_72/{case}…        已分類（日期 yyMMdd _ case 數）
  Die/261022_34/{case}…
  261028_169/{case}…           待處理的上傳
  261029_92/{case}…
```

- case 資料夾內有檔名 `…-Tooth_{FDI}.stl`（FDI 兩位數，第一位 1–8、第二位 1–8）→ **Die**；沒有 → **IOS**。`…-45-crown_cad.stl` 這類雖帶牙位編號，但不是 Die。
- `{case}/{case}/{files}`（處理不好留下的重複資料夾，可多層）會先展平成 `{case}/{files}`。
- `IOS`、`Die` 不存在會自動建立，權限 777；同日期已有批次資料夾就併入並改成實際 case 數。

## 它做什麼、不做什麼

- 只用「搬移資料夾」整理，不複製、不刪內容；目的地已有同名 case 就略過並回報，**不會覆蓋**。批次資料夾搬空後才會移除。
- 每次執行都在 `<dataset>/.scan-origin/moves-<時間>.json` 記下所有變動，需要還原時可照著手動搬回。
- 沒有「還原」按鈕，也不處理 IOS／Die 裡既有 case 的重新分類。

## 建置與測試

```bash
go build -o scan-origin .
go vet ./... && go test ./...
```

開發紀錄在 `ai-log/`（給 AI 讀，英文）。需求文件在 `doc/`。

## 發行（維護者）

版本號在 `internal/version/version.go`（三段式 semver）。改版、commit 後：

```bash
go run ./tools/release        # dist/ 產生三個平台壓縮檔與 SHA256SUMS.txt（有未 commit 的改動會拒絕）
go run ./tools/npmpack        # dist/npm/ 產生四個 .tgz（三個平台包 + 啟動器包），並印出 npm publish 指令
```

兩支工具都不會上傳。npm 要先發三個平台包、最後發 `@cadcam/scan-origin`（需先有 `@cadcam` 組織並 `npm login`）；GitHub Release 則自行把 `dist/` 的壓縮檔與 `SHA256SUMS.txt` 附上。

## 授權

MIT，見 `LICENSE`。
