# scan-origin

依檔名把牙齒掃描 dataset 的 case 分成 **IOS（口內掃描）** 與 **Die（桌面掃描，石膏模切 Die）**，並統計數量。Go 標準庫 + 原生 HTML/JS/CSS，單一執行檔。

## 安裝

發行檔提供 Windows x64、macOS Apple Silicon、Linux x64。

```bash
npm i -g @cadcam/scan-origin     # 需要 Node.js 16+，npm 會自動挑對應系統的執行檔
```

或到 GitHub Release 下載壓縮檔（`scan-origin_<版本>_<windows|darwin|linux>_<amd64|arm64>`，附 `SHA256SUMS.txt`），解壓後直接執行。目前沒有數位簽章與 Apple 公證，第一次執行 macOS（Gatekeeper）或 Windows（SmartScreen）可能跳警告。

## 使用

在 dataset 電腦上執行：

```bash
scan-origin                      # 第一次：以目前資料夾為 dataset；之後：不論在哪執行，都用上次的資料夾
scan-origin -here                # 強制以目前資料夾為 dataset，並記住它
scan-origin /path/to/dataset     # 或明確指定資料夾（同樣會被記住）
```

選擇順序：命令列資料夾 > `-here` > 上次記住的資料夾 > 目前資料夾。每次啟動伺服器後會把實際使用的資料夾記在使用者設定資料夾的 `scan-origin/last.json`（Windows `%AppData%`、Linux `~/.config`、macOS `~/Library/Application Support`）。若用的是上次的資料夾、且和目前資料夾不同，啟動時會在 stderr 提醒；上次的資料夾已不存在則退回目前資料夾並警告。`-here` 不能和資料夾參數同時使用。`-tree` 用同一套規則找資料夾，但不會改寫記錄。

啟動後會印出含 token 的網址，用另一台電腦的瀏覽器開啟（例如 `http://192.168.1.20:8765/?t=…`）：

- **Dataset 目前狀態**：已分類總計、IOS 數、Die 數、待處理數，以及每個上傳日期的 IOS／Die 數。
- **各牙位（FDI）數量**：每個 FDI 牙位在 IOS、Die 各有幾個 case，以及總量（見下方「牙位統計」）。
- **匯出 Excel**：頁面右上角的「匯出 Excel」下載 `.xlsx`（見下方「匯出 Excel」）。
- **整理待處理的上傳**：先按「預覽整理結果」看每個 case 會被判成 IOS 或 Die，確認後才「執行」。

選項：`-host`（預設 `0.0.0.0`；只給本機用請設 `127.0.0.1`）、`-port`（預設 8765，被占用且未指定時改用隨機埠）、`-no-open`（不在本機開瀏覽器）、`-here`（見上）、`-v`／`-version`（印出版本後結束，可用來確認 npm 裝到的是哪一版）。結束用 Ctrl+C。

> 預設監聽所有網卡，Windows 第一次執行可能跳防火牆提示。首頁與 API 都需要啟動時產生的隨機 token，**知道含 token 的網址就能整理這個資料夾**，請勿外流；每次啟動 token 都會改變。沒有 TLS，請只在可信任的內網使用。

### 列出檔案結構（`-tree`）

要把 dataset 的真實路徑清單貼給 AI 或拿來開發時，ssh 到 dataset 那台執行（資料夾依上面的規則決定，要列目前資料夾就加 `-here`）：

```bash
scan-origin -tree                      # 列出目前資料夾（只有名稱，沒有內容），並送到剪貼簿
scan-origin -tree -depth 3 -sample 3   # 精簡版：最多 3 層、每個資料夾最多 3 項
scan-origin -tree -o tree.txt          # 存成檔案（再用 scp 取回）
```

- 一次性指令：印完就結束，不啟動伺服器，也不會動到 dataset。
- 輸出是縮排樹狀，資料夾以 `/` 結尾；`.scan-origin/` 不列出。被 `-sample` 或 `-depth` 截斷處會標 `... (+M more)` 或 ` ...`，結尾的統計也會註明「truncated」。
- 選項（只能搭配 `-tree`）：`-depth N`（只列 N 層）、`-sample N`（每個資料夾最多 N 項）、`-dirs-only`（只列資料夾）、`-o 檔案`（寫到檔案）、`-no-copy`（不送剪貼簿）。
- **剪貼簿**用 OSC 52：由你**本機**的終端機把內容放進本機剪貼簿，所以 ssh 時也能用。Windows Terminal、iTerm2、kitty、WezTerm 支援；tmux 裡要設 `set -g set-clipboard on`；GNU screen 不支援。只有輸出到終端機時才會送（`> file` 不會混入控制碼）。
- 清單太大（約 70 KB 以上）時為了避免被終端機靜默丟棄，不會送剪貼簿並會提示；請加 `-depth`／`-sample`／`-dirs-only` 縮小，或改用 `-o`。貼上是空的就是終端機不支援，同樣改用 `-o`。

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

## 牙位統計

只統計已分類在 `IOS/`、`Die/` 底下的 case（待處理批次不算），牙位依 case 所在的類別決定：

- **IOS** case：`…-35-crown_cad.stl` 的 `35`（`…_transform.stl` 不算，避免重複）。
- **Die** case：`…-Tooth_36.stl` 的 `36`（Die case 裡的 `…-36-crown_cad.stl` 不看）。
- FDI 兩位數，第一位 1–8、第二位 1–8，不分大小寫。一個 case 有多顆牙（如牙橋）時，每個不同的牙位各算一次，所以牙位總量可能大於 case 數。
- 找不到牙位檔的 case 不計入，頁面會列出前 50 個。
- 需要列出每個 case 的檔名，所以頁面先顯示一般統計，牙位表稍後載入（API：`GET /api/teeth`，純讀取）。

## 匯出 Excel

頁面右上角按「匯出 Excel」，瀏覽器會下載 `scan-origin-<日期>-<時間>.xlsx`，內含四張工作表：

- **摘要**：已分類 case 數、牙位數、找不到牙位檔的 case 數（分 IOS／Die／合計），以及待處理 case 數。
- **各上傳日期**：每個日期的 IOS、Die、小計，最後一列總量。
- **各牙位**：每個 FDI 的 IOS、Die、小計，最後一列總量。
- **Case 清單**：每個已分類 case 一列（類別、批次、case 名稱、牙位、牙位數），可在 Excel 自行篩選。

統計規則與上面相同（只算已分類的 case）。純讀取，不會動 dataset；不依賴第三方套件（自行寫入 `.xlsx`）。API：`GET /api/export.xlsx`（要 `X-Token`，所以是頁面用 fetch 下載，不能直接貼網址）。

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
