# T44-R1 · Web 事件传播、布局约束及实际可交互区域

#660 / #563；K02 交互边界，K03 事件、几何与布局算法，K01 任务观察，K05 评价，K10 响应成本。冻结 b9b162da52e683f40bebfb58ddda2288cd5a365f。HC-04 的 Web 有限机制，复用 T24c 的任务/焦点/输入、T25b-1 字符身份、T08-C04 状态通知及 T35-3 G4 的请求/权限。以下是原创纸面合同与算例，不是全平台实现或实际浏览器测量，所有目标 NOT_RUN。

## 1. 平台事件、应用意图和业务效果分离

同一业务命令可来自键盘、触摸、指针或辅助技术激活，不把某一种物理输入当唯一业务入口。输入事件携带平台上下文；界面将其解释为有权业务意图候选（含稳定对象身份），服务端/权威再检查当前规则并执行。事件传播停止不等于取消业务命令、网络请求或已提交效果；事件的 isTrusted 也不是用户权限或业务确认凭证。优先原生语义控件及其适用激活合同，不能假定手写 div click 等价于键盘/焦点/辅助技术支持。

以普通 DOM 树 R（根）→P（父）→T（目标），无 shadow、无嵌套 dispatch、无监听器变更，教学事件 E 的 bubbles=true、cancelable=true；所有监听器正常同步返回，只有 T 的两个非 capture 监听器按注册次序 T1、T2。注册 R/P capture 为 Rc/Pc，T capture 为 Tc，P/R 非 capture 为 Pb/Rb。
| T1 行为 | 预期监听顺序 | canceled flag |
|---|---|---|
| 无特殊操作 | Rc,Pc,Tc,T1,T2,Pb,Rb | false |
| stopPropagation | Rc,Pc,Tc,T1,T2 | false |
| stopImmediatePropagation | Rc,Pc,Tc,T1 | false |
| preventDefault，非 passive | Rc,Pc,Tc,T1,T2,Pb,Rb | true |
| preventDefault，passive 或 E 不可取消 | Rc,Pc,Tc,T1,T2,Pb,Rb | false |

表中没有其它取消操作；不可取消变体仍 bubbles=true。目标上 capture 与非 capture 都可执行，eventPhase 为 AT_TARGET；bubbles=false 普通树仍可有祖先捕获及目标两类监听，但没有祖先冒泡。DOM Standard §2.2/2.9 定义传播/立即停止/取消三个标志及 passive 条件，并按阶段遍历事件路径；这些条件不等同于所有默认动作在所有监听之后运行，实际控件激活可有 pre-activation 与取消恢复。教学自定义事件的 canceled flag 只表明可取消标志是否设置，并不假设一定有浏览器默认动作。DOM dispatch 返回 false 表明事件被取消，不是命令执行失败或所有 listener 成功的收据。

## 2. 事件委托、动态身份与监听所有权

target 是按当前事件路径/封装规则暴露的事件目标，currentTarget 是当前调用监听器的对象，派发后为 null；Shadow DOM 可重定向目标并限制可见路径，composed/bubbles 独立，不假设外部委托器永远能读取内部按钮。对普通列表可在受控容器监听并按实际元素找到获权操作对象，必须限制匹配属于该容器的有效项目；不能按数组 index、显示文本或任意 data 属性直接执行敏感操作。例：行 [A,B] 的 B 删除按钮展示时 index=1，重新排序 [B,A] 后按旧 index=1 会删 A；绑定稳定 B 的身份并核当前存在/版本/权限，才不把显示位置当业务对象。即使监听函数相同，也不能合并“选择项目”与“永久删除项目”的业务效果。

浏览器在一次 invocation 取监听列表快照，之后添加者不加入该 invocation；被移除的条目仍被排除，不推广为整场 dispatch 全局快照；一次性 listener 在调用前移除。重复挂载/卸载应让拥有者登记并释放同一监听与资源；DOM removeEventListener 匹配 type/callback/capture，或按注册时 signal 等实际机制。匿名新闭包不能凭相同源码文字移除旧 callback。区分同步事件结束与异步提交：在 await 后使用原 event.currentTarget 可能已 null；先捕获必要稳定身份，异步结果仍需按真实活跃界面、请求身份及权威授权校验（T35-3），不是把整个 DOM 节点永久保留在业务对象里。

## 3. 布局：尺寸分配不是比例常数

布局、绘制、合成及实际命中分别回答几何、像素呈现、组合与输入目标，不要求所有平台具有相同内部流水线。CSS flex 是基于可用空间、基准尺寸、grow/shrink 权重、min/max 约束的分配机制。教学限定水平单行 flex、内宽300 CSS px、无边距/边框/padding/gap、两个非替换 item A/B，flex base size 200/200、shrink factor 各1、显式 min A=180 B=0、无 max 约束且内容允许 B 缩小。第一次负空间−100，按 scaled shrink 权重200:200 得150/150；A 被最小值 clamp 至180，冻结 A 后重新计算，B=120；总300。若 B 也要求 min=150，则至少330，无法靠 shrink 保证装入300；允许溢出、换行、滚动或改变设计必须由任务/可访问合同选择，而非自动隐藏必需控件。

反例：无 min/max 限制，base 200/100、shrink 均1，内宽240，差额−60 按200:100 分配为−40/−20，得到160/80；不是每项减30。正自由空间例 base100/100，grow1/2，内宽350，得150/200。在满足该简化模型权重和边界时，实际算法每轮至少冻结一个 item 而终止；不能由此推导整个浏览器布局复杂度、所有内容都能无溢出或无限百分比约束已解。Flexbox 2025-10-14 CRD §9.7 描述缩放、钳制、冻结；§4.5 的自动最小尺寸随 overflow/内容及具体条件变化，不等同固定 min=0。案例显式指定 min，不以全局 min-width:0 作为通用正确性修复。

## 4. 观察几何与命中：坐标空间及版本

CSSOM View 2025-09-16 WD 的 getClientRects 返回布局片段边框矩形并应用适用变换；getBoundingClientRect 是其包围矩形，返回对象不是 live。教学行内元素两个有效片段在同一坐标系 R1=[10,60)×[20,30)，R2=[10,30)×[30,40)，包围矩形=[10,60)×[20,40)。点(50,35) 在包围矩形内却不在任一片段内，因此单一矩形不能作为真实命中充分证明；裁剪、遮挡、变换、pointer-events、滚动和平台命中规则还会改变实际可交互区域。此处半开集合是教学几何约定，不规定 CSS 边界像素命中。

教学无变换、固定滚动坐标关系：元素视口 top=80，页面 scrollY=200，则文档 top=280；指针 clientY=90 时同视口差10。错误将文档 y=290 与 viewport top80 比得到210。若在不同帧/滚动代际测量，须重新核坐标与布局版本；缩放、设备像素比、visual/layout viewport、嵌套滚动及 transform 不能只用一个常数修补。文字逻辑顺序、视觉 BiDi、DOM 顺序、焦点遍历亦分开；沿 T24c/T25b-1 按真实书写方向、输入和可访问要求适配，未读完整 BiDi 或各平台 collation 不宣称支持。

T44-RV-01 限定补充：上述普通非退化片段的包围矩形解释不能概括所有 getBoundingClientRect 结果。实际重读 [CSSOM View WD 2025-09-16](https://www.w3.org/TR/2025/WD-cssom-view-1-20250916/#dom-element-getboundingclientrect) L941–952（2026-11-27）：先取 getClientRects 列表；空列表返回 x/y/width/height 全零；若每个矩形至少一个维度为零，则返回列表首项；否则包围至少一个维度非零的所有矩形（因此在此分支可含线段，并非只选正面积）。教学矩形以 (left,top,right,bottom) 表示：点 (100,100,100,100) 与正常矩形 (10,20,60,30) 同时存在时，包围结果为后者而非 (10,20,100,100)；若另有线段 (80,5,80,25)，则包围为 (10,5,80,30)。若只有点与线段且点列首，返回该点；空列表为全零。矩形仍非 live，包围区域不等于真实命中/可见区域，原两个非退化片段与滚动反例保留。此为指定日期 WD 的局部规则和原创纸面反例，未执行浏览器，不推广到任意平台。

## 5. 条件选择与有限评价（NOT_RUN）

静态小界面先选原生控件及简单布局；重复动态行且共享交互机制才考虑委托；复杂视图按有权任务和独立业务状态拆分；只有真实长列表/帧预算需要才选择虚拟化与测量，虚拟化需核身份、焦点、滚动恢复和屏幕阅读。固定布局算例不证明可用性。独立预期：表中五种传播/取消、非冒泡变体、重排 B 的身份、异步 currentTarget、缩放及冻结算例、两个片段的包围空白和滚动坐标反例；实际平台还需多输入模式、不同视窗/文字、遮挡与真实任务的独立观察。本阶段不运行浏览器/产品/框架、性能或用户实验。

实际读取旧源 T24c b5dc06f0411a3f0ce3cc6f939e3d4ed31e93152a L5–71；T25b-1 ddab94c92d9fdfe727fba9fc6bf1054161d602d3 L63–77；T08 1656be94dc3eb1764e7bfa09aa5fb6bb962b38c0 L29–44；T35-3 6d3d81acd9fa952ebc3f06e1af9d5e4250822f89 L45–71。2026-11-27 实际读 [DOM Living Standard](https://dom.spec.whatwg.org/#events) 更新2026-11-25，§2.2 L442–475、§2.7 L643–648/701–734、§2.9 L813–856/875–941；[CSS Flexbox Level1 CRD 2025-10-14](https://www.w3.org/TR/2025/CRD-css-flexbox-1-20251014/) §4.5 L753–758/894–899、§9.7 L2284–2317（实际读可定位同日期 TR）；[CSSOM View WD 2025-09-16](https://www.w3.org/TR/2025/WD-cssom-view-1-20250916/) L912–923/941–952（实际读同日期 TR）。引用的是规范局部机制，不声称浏览器实现全部符合或 WD/CRD 已为最终推荐；算例原创，HC-04 整体 SPECIALIST U，无新 qualified Case/Claim/Rule。
