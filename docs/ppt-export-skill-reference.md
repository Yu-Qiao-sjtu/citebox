# PPT 导出功能参考：wisp journal-club-ppt 技能链拆解

本文为 CiteBox「一键导出汇报 PPT（中+英；英文 Arial、中文微软雅黑）」需求的实现参考，
拆解对象是本地 AI 工作台 wisp（wisp-science，Tauri 桌面应用）的 PPT 制作能力。

- 源码位置：`C:\Users\13260\Documents\ChatGPT\issue`（wisp 仓库本地克隆）
- 技能许可：各技能 frontmatter 标注 Apache-2.0
- 相关版本：离线 PPTX 预览见 wisp `v0.17.0` release notes；`journal-club-ppt` 技能见 `v0.20.0`

## 一、总体结论

wisp 里**没有工程化的"一键导出 PPT"代码**。它的 PPT 制作分三层：

| 层 | 组成 | 作用 |
|---|---|---|
| 技能层 | `skills/journal-club-ppt/SKILL.md`（纯 prompt，无脚本） | 定义"论文 → 汇报 PPT"的 9 阶段工作流、硬约束、质量门 |
| 运行层 | 技能引擎（`use_skill`）+ 持久 Python kernel + 基础工具（`view_image` 等） | 加载技能、执行 PDF 解析/裁图/pptx 生成（模型现场写代码） |
| 预览层 | vendored `@aiden0z/pptx-renderer@1.2.4`（`ui/vendor-build`） | 前端离线预览生成的 .pptx，只读不生成 |

对 CiteBox 的含义：**wisp 可借鉴的是技能里的"内容生产方法论"**（大纲结构、
图表选择规则、质检清单），而不是它的运行方式——它依赖持久 Python 运行时和
模型在环，结果不可复现；CiteBox 的一键导出应走"确定性模板 + 数据填充"。

## 二、技能引擎机制（骨架，可借鉴）

- **SKILL.md frontmatter**：`name` / `description`（触发条件的自然语言描述，
  含中英文触发词）/ `fold_cue`（何时折叠）/ `license`。
- **`use_skill` 工具**（`crates/wisp-skills/src/tool.rs`）：把 SKILL.md 渲染进
  模型上下文；若技能目录带 `kernel.py`，输出末尾追加「Python Kernel Sidecar」
  一节，给一条 `exec(...)` 行，由模型执行后把 helper 函数定义进持久 kernel。
  wisp 不做宿主自动注入，靠这条约定加载。
- **持久 Python kernel**（`crates/wisp-runtime/src/kernel.rs`、`manager.rs`）：
  跨 cell 保留变量（论文大纲、图表清单、裁图路径、slide plan 都留在 kernel 里
  复用）；依赖用 uv venv + `uv pip install` 管理（`env.rs`）。
- **`view_image`**（`crates/wisp-tools/src/image.rs`）：本地图片 → data URI 进
  视觉模型；PDF 二进制 `read` 工具不解析，一律走 Python（`read.rs` 扩展名表
  里的 pptx 只影响文本读取分类）。

## 三、journal-club-ppt：论文 → 汇报 PPT 的完整调用链

文件：`skills/journal-club-ppt/SKILL.md`（仅此一个文件）。

定位：把一篇科学论文 PDF 变成组会/journal club 用的"论证驱动"汇报 PPT
（10–30 页），讲论文的逻辑，不是分节摘要+截图粘贴。默认中文、关键术语保英文。

### 调用链（它"调用了哪些东西"）

1. `use_skill(journal-club-ppt)` —— 加载工作流与约束。
2. **读论文**：借 `pdf-explore` 风格的 helper（见 §四.1）在持久 kernel 里解析
   PDF：`pdf_outline()` 拿书签目录，`pdf_pages(mode="text")` 按需取章节文本，
   `mode="image", dpi=200` 渲染页面 PNG。
3. **看图**：`view_image` 看渲染页，检查图版式、面板标签、图例、坐标轴、比例尺。
4. **建 figure_ledger**（Python 里维护的图表台账 CSV）：每行记录
   `figure_id / panel_id / page / caption_summary / paper_claim_supported /
   visual_type / include_decision / reason / crop_path`。
5. **裁子图**：Python(Pillow) 按 200–300 DPI 渲染后裁剪，仅保留支撑论证的面板，
   命名如 `figures/Fig2c_p06_validation.png`；保留轴/图例/比例尺/面板字母。
6. **模型现场写 Python 生成 .pptx**（仓库无固定脚本、无预置 python-pptx 依赖声明）。
7. **前端离线预览**：`@aiden0z/pptx-renderer`（`ui/vendor-build/package.json`，
   由 `ui/vendor-src/office-build.json` 固定版本+SHA256 管理）。

### 硬约束（8 条，摘译）

1. 先读懂论证再设计幻灯片（中心问题/缺口/假设/证据链/最终结论）。
2. 必须有作者介绍（一作、通讯、单位、方向、为什么这个团队做这个工作）。
3. 必须有背景介绍（领域问题 → 未解缺口 → 本文为什么重要 → 本文问题）。
4. 除非用户要求全自动，先确认大纲再填页；大纲必须证据驱动并标注计划用的图版。
5. 只用正文 Figure 的图，禁止网图/库存图/AI 图/作者照片/期刊 logo/补充图/Extended Data。
6. 按逻辑选子图，不整页截图、不倾倒所有面板。
7. 必须以评估收尾（优势、局限、未解问题、可能的后续实验）。
8. 页数是硬性质量门：10–30 页（含标题与讨论页）；短文 10–12、标准论文 14–18、
   复杂多组学/方法重型 19–24、确有必要才 25–30。

### 标准 12 段大纲

标题页 → 一页结论（核心 claim + 为什么重要）→ 作者/团队 → 背景 I（领域问题）
→ 背景 II（知识缺口）→ 研究设计/方法总览 → 结果 1..N（每页配选中面板）→
整合模型/结论 → 优势 → 局限与讨论（收尾给 2–4 个组会讨论问题）。
超 12 页时只允许"拆分复杂结果链"，不允许注水；每个新增结果页必须回答
"这页让哪个 claim 更可信了"。

### 结果页结构模板

- 标题：一句论点（如"单细胞图谱揭示 X 细胞群扩增"），不是 topic 标签（"Figure 2"）。
- 左/上：选中的图；右/下："怎么读" + "证明了什么"；页脚或小字标注来源面板 ID（`Fig. 3b`）。
- 演讲者备注：实验设置、每个面板看什么、坐标轴怎么解读、注意事项。

### 子图选择打分（定性）

`include_score = 与核心claim的中心性 + 填补关键缺口 + 方法解释价值 + 视觉可读性 − 冗余 − 过度细节`，
并列出反模式（因为存在所以全用、整页截图、裁掉坐标轴/图例/比例尺、同一面板重复用等）。

### 交付物与自检

- 交付：`deck_outline.md`（逐页计划）、`figure_ledger.csv`、`figures/`、
  `journal_club_ppt.pptx`。
- 交付前自检 checklist（10 项）：页数达标；首个结果页前有作者/背景/方法铺垫；
  每个结果页有论点标题；所有图均来自正文裁图；无补充图/网图；面板可读且
  标签图例齐全；核心逻辑可从页面序列读出；优势/局限具体不空泛；重要面板
  有读图说明。

## 四、配套技能（对"补充数据"部分直接有用）

### 1. pdf-explore —— PDF 无上下文洪泛阅读

文件：`skills/pdf-explore/SKILL.md` + `kernel.py`。依赖 `pypdfium2`（+ `pillow`）。

| helper | 用途 | 返回 |
|---|---|---|
| `pdf_outline(path)` | 先试书签目录 | `[{page, heading, level}]` |
| `pdf_pages(path, pages, mode="text")` | 按页取文本（≤5 页可直接打印） | `[{page, text, n_chars}]` |
| `pdf_pages(path, mode="image", dpi=200, pages)` | 图版/扫描页渲染 PNG（缓存于 `.cache/pdf-explore/`，配 `view_image`） | 每页 PNG |
| `mode="auto"` | 未知 PDF：有文本层走文本，无则翻图像 | — |

解析一次、磁盘+内存缓存；大输出写文件再 `read`，避免撑爆上下文。

### 2. figure-style —— 出版级单图规则 + CJK 字体方案

文件：`skills/figure-style/SKILL.md` + `kernel.py`。

- `apply_figure_style(frame, font, sizes=(8,7,6), grid)`：设 matplotlib rcParams
  ——角色映射的字号阶梯、朝外刻度、无框图例、300 dpi、Type-42 字体内嵌。
- helper：`focal_palette` / `bar_with_points` / `strip_with_median` /
  `end_of_line_labels` / `panel_letter` / `set_frame` / `panel_crops`。
- 正确性规则（§1–§3）：排除数据不得进入统计量、claim 式标题必须逐行验证为真、
  每面板标注 n 与固定变量、一个定量 claim 全文只有一个权威数值。
- 标签经济（§2）：下限=可识别性（删掉就看不懂的标签不可去），上限=每面板叙述性
  标注 ≤2–3 个。
- **CJK 字体回退清单**（出图防"口口"）：按 OS 探测字体文件并注册——
  Windows：`msyh.ttc`（微软雅黑）→ `simhei.ttf` → `simsun.ttc`；
  macOS：PingFang SC → Hiragino Sans GB → STHeiti；
  Linux：Noto Sans CJK SC → WenQuanYi Zen Hei → Source Han Sans SC。
  CiteBox 的图表/报告字体策略可直接沿用这份优先级。

### 3. figure-composer —— 多面板组合 + 对抗式评审

文件：`skills/figure-composer/SKILL.md` + `kernel.py`。

- 输入三要素：一句话 claim、目标宽度(mm)、具体数据路径。
- outline schema：12 列网格，`panels[{letter, role, row, col, colspan,
  chart_family, message, data_path, ask}]`；面板 a=概念钩子（schematic），
  面板 b=主要证据。
- 流程：outline → 每面板一条渲染指令（批量委托或顺序 Python 渲染）→
  `compose_figure()` 组装 → Pillow `compose_crops()` → `view_image` 自检
  接缝/裁切标签/留白/面板字母 → 独立评审者对抗式审图 → 只重做受影响面板。
- 收敛条件：≤3 轮，或无 blocker 且 major 发现 ≤2。
- 边界：Python 里不调模型、`view_image` 只看本地文件。

### 4. paper-narrative —— 全文叙事排序（简）

`skills/paper-narrative/SKILL.md`：产出 `arc`（主图顺序）、`figure_moves`、
`missing_panels`、`kill_list`、`boldest_defensible_fig1`，用于判断 Figure 1
钩子力与图表顺序。PPT 场景主要用于结果页排序校验。

## 五、映射到 CiteBox 的实现建议

### 资产对照

| wisp 的做法 | CiteBox 已有/对应 |
|---|---|
| 持久 Python kernel 现场写代码 | ❌ 不引入；改为确定性模板填充 |
| pdf-explore 解析 PDF | ✅ 已有 `pdf_text` 入库 + pdf.js 资产；裁图可用后端 pdfium 或直接取图片库 |
| figure_ledger | ✅ 图片库已含子图拆分（a/b/c/d/e/f）、tag、图片解读笔记 |
| 每结果页 claim + 读图说明 | ✅ AI 伴读/批量图片解读（#40）产出的 per-figure 解读可直接填页 |
| 文献出处标注 | ✅ 文档大库命名规则（时间-通讯作者+文件名）→ 页脚引用与来源页 |
| 12 段大纲 / 自检清单 | 作为导出模板的固定结构与导出前校验规则 |

### 建议流水线（确定性，中英双语）

1. **选源**：用户选 1 篇文献（或 collection），后端汇总元数据（标题/期刊/年份/
   DOI/作者与通讯作者）、摘要、`pdf_text` 章节线索、图片库（图/子图 + 解读笔记）。
2. **内容中间层**：生成一份 deck JSON（逐页：类型/标题 claim/正文要点/图片引用/
   双语字段/备注）。首版可由 AI 伴读模型起草并落库，允许用户在 UI 改——这一层
   与「文档式 HTML 报告」需求共用，一份 JSON 两个渲染出口。
3. **渲染**：前端 pptxgenjs（vendored，仿 wisp `office-build.json` 固定版本+SHA256，
   不引构建步骤）或 Go 侧 OOXML 模板填充，产出 `.pptx` 落盘。
4. **字体（本需求硬约束）**：在模板主题里一次性设
   `fontScheme: latin=Arial, ea=微软雅黑`，全 deck 继承；不要逐文本框设字体。
   注意 pptxgenjs 的 `fontFace` 只写 `<a:latin>`，中文字符仍会回落主题 ea 字体
   （默认等线）——需补 `<a:ea typeface="微软雅黑"/>` 或直接改主题。
5. **图片**：优先取图片库已裁子图（含 tag 与解读），避免运行时裁 PDF；确需裁剪
   时后端做。每图带来源标注（`Fig. 3b`、文献、时间-通讯作者）。
6. **导出自检**（沿用 journal-club-ppt 的 checklist 裁剪版）：页数范围、每结果页
   有 claim 标题、图片有来源、双语字段齐全、（可选）解包校验字体 XML 命中。
7. **预览/后续**：可选仿 wisp 用 `@aiden0z/pptx-renderer` 做离线预览，或首版
   仅提供下载。

### 与 TODO 的关系

对应 `TODO` →「汇报与导出」节的：一键导出 PPT（中+英，文档式报告+补充数据组合，
英文 Arial、中文微软雅黑）；「文档式 HTML 报告（中+英）」共用同一 deck JSON 中间层。

## 六、实证案例：Codex（ChatGPT 桌面版）实际制作汇报 PPT 的链路还原

以下是对 2026-09-09 组会文献汇报 PPT（`D:\New-PHD\组会\20260911-qy-paper.pptx`，
CAR-AM 论文，最终 11 页）制作过程的实证还原。证据来源：Codex Desktop 会话记录
（`~/.codex/sessions/2026/09/09/rollout-*.jsonl`，cwd=`D:\New-PHD\组会`，
模型 GPT-5，会话内含 223 次图片视觉输入）、工作目录产物（16 个版本迭代、
validation JSON、QA 渲染图）。

### 阶段 A：对话驱动的内容生产（先文字，后版式）

1. **PDF 内容理解**：用户逐模块提问（图1 的论述逻辑？结果1 讲了什么？），
   Codex 加载自带 pdf 技能（`~/.codex/plugins/cache/openai-primary-runtime/pdf/`），
   用 poppler（`pdfinfo`/`pdftotext`）+ pypdf 抽文本，把结果页渲染成 JPG 后
   `view_image` 逐页看图，回答论文逻辑问题。
2. **文字稿落盘 docx**：用户要求"把全部结果按这个逻辑梳理成 docx"→ Codex 加载
   documents 技能，生成 `build_car_am_results_docx.py`（python-docx）构建，
   再用 Word COM 把 docx 转 PDF、`pdftoppm -png -r 120` 渲染每页 PNG、
   逐页视觉检查（tmp/docx_qa1..5 五轮），修正后再渲染。
3. **补充内容迭代**：用户继续追问（肺泡巨噬细胞 marker、科学问题/创新性/
   临床问题），全部并入文字稿。

### 阶段 B：PPT 装配（构建-渲染-看图-修正循环）

4. **加载 presentations 技能**（Codex 内置，含 style_guidelines.md 反 AI 腔
   写作规范、template_following 模板遵循规范）。
5. **结构化检查既有 deck**：`inspect_deck.mjs` 用 `@oai/artifact-tool` 的
   `PresentationFile.importPptx` 读入 pptx，导出每页 layout JSON 和结构
   ndjson，并计算 SHA256 基线。
6. **渲染基线**：PowerPoint COM（`Presentations.Open → SaveAs PDF(格式32)`）
   → poppler `pdftoppm -png -r 110` → `view_image` 逐页看原版 6 页。
7. **写构建脚本**：`build_results_deck.mjs`（Node + @oai/artifact-tool）——
   核心数据是一张映射表：`{figure: 2, title: "结果1 级联靶向脂质体纳米药物
   原位生成CAR-AM", page: 4}` × 7 条（论点式标题 → PDF 图所在页）；
   从 PDF 页裁结果图、复用模板版式（slideLayout7.xml）、标题用预渲染 PNG
   （titles/title-N.png，保证字体样式）、字体设 Arial + 微软雅黑。
8. **16 版迭代**：每版 = 构建 → 全页渲染 PNG → 逐页 view_image 检查
   （空白标题、溢出、图版错位等都这样抓出来）→ 修 → 存
   `output/pptx/…-vN.pptx`。
9. **程序化终检**：`.codex-finalizer-pptx/*.validation.json`
   （schema `presentation-finalization.v1`）：
   - 包完整性：ZIP 部件、关系数、页数（11）、画幅 13.33×7.5in；
   - **字体策略：枚举全部文本 run，observed {Arial: 136, 微软雅黑: 4} →
     passed**——"英文 Arial、中文微软雅黑"是程序化验证过的，不是口头约定。
10. 交付后在 WPS 里人工微调保存（所以成品 app.xml 显示"WPS 演示"）。

### 该链路对 CiteBox 的直接结论

- **先内容后版式**：对话/解读产出结构化文字稿（对应 CiteBox 的 deck JSON
  中间层），再装配版式——两阶段解耦。
- **质量来自"构建→渲染→逐页看图→修复"循环**（本次 16 版），而非一次成型；
  CiteBox 至少要做"渲染每页 PNG + 程序化检查"，AI 视觉检查可作为可选增强。
- **终检必须程序化**：字体策略逐 run 统计（Arial/微软雅黑）、页数、包完整性
  ——这套 validation JSON 的 schema 思路可直接移植到 CiteBox 导出器。
- **论点式标题 + 图页映射表**是 deck 的最小数据结构
  （figure → claim 标题 → 来源页），CiteBox 图片库的 tag/解读笔记天然就是
  这张表的来源。
