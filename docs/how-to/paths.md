**How-to** · [Workflow](README.md) · [Set up](setup.md) · [Rules](routing.md) · [Paths](paths.md) · [Incidents](incidents.md) · [Connect](connect.md) · [Fix it](troubleshooting.md)

**English** · [Русский](ru/paths.md)

*Admin · escalation paths*

# How L2 is allowed to hand work to L3.

A support tier on a team is only a badge until you add a **path**. The path
is the allow-list: “from this team, the incident screen may offer that team.” You add it on
the **source** team (the one that will click the button), under **Escalation paths** →
**Outgoing paths** → **Add path**.

[Why this is separate from routing](#why) | [Which tiers can point where](#allowed) | [Where to click](#where) | [Example: L2 → L3](#ex1) | [Example: NOC chain](#ex2) | [Example: shared L3](#ex3) | [Empty picker / missing button](#empty)

<a id="why"></a>

## 01 · Routing vs a path

|  | Routing rule | Escalation path |
| --- | --- | --- |
| Question it answers | Which team owns a **new** alert? | Where may that team **send** an open incident? |
| Where you configure it | **Workspaces** → project → **Routing rules** | **Teams** → source team → **Escalation paths** |
| Typical target | L2 or NOC | The next tier (L3, or L2 from NOC/L1) |
| If missing | Alert never becomes an incident | No **Hand off to L3** / escalate button, or the picker is empty |

**Bounce** is not a path you
configure. L3 clicks **Bounce to L2** and Aegis returns the incident to the previous owner.

<a id="allowed"></a>

## 02 · Allowed from → to

The **To team** select only lists teams whose tier is legal for the source.
Both teams need a support tier set (**Save tier** on team settings).

| Source team tier | You may add a path to | Button on the incident |
| --- | --- | --- |
| NOC | L1 or L2 | Escalate to L1 / Escalate to L2 |
| L1 | L2 | Escalate to L2 |
| L2 | L3 | Hand off to L3 |
| L3 | Nothing outgoing | Bounce to L2 (not a path) |

> **No valid target teams for this tier**
>
> That empty-state on **Add path** means there is no L3 (or L2/L1) team yet, the only
> candidate already has a path, or you need **Allow team from another workspace** because
> the target lives elsewhere. Create the target team and set its tier first.

<a id="where"></a>

## 03 · Where to click

1. **Open the team that will click Hand off**
   Sidebar **Teams** → **Platform L2** (not L3). Scroll to
   **Escalation paths**. You will see **Outgoing paths** (empty at first) and
   **Incoming paths** (paths other teams already point at this team).

2. **Add path**
   Click **Add path**. Optionally check **Allow team from another workspace**.
   Choose **To team**. Confirm. Toast: **Escalation path added**.

3. **Confirm on both sides**
   L2 **Outgoing paths** shows To team = Platform L3. Open Platform L3 —
   **Incoming paths** shows From team = Platform L2.

<a id="ex1"></a>

## 04 · Example 1 — Platform L2 → Platform L3

Minimum for most projects. Teams already exist with tiers L2 and L3 in
workspace Platform.

#### Add path · on team **Platform L2**

| Field | Value |
|-------|-------|
| Allow team from another workspace | Off |
| To team | Platform L3 |

On an incident owned by Platform L2, the engineer sees **Hand off to L3**
and can pick Platform L3. Aegis pages whoever **On call now** is on Platform L3. If L3
has no schedule, handoff fails with a message to open **Shifts**.

<a id="ex2"></a>

## 05 · Example 2 — NOC → L1 → L2 → L3

Four teams in workspace Platform: Platform NOC, Platform L1, Platform L2,
Platform L3. Add **three** paths (NOC may also skip L1).

#### Path 1 · on **Platform NOC**

| Field | Value |
|-------|-------|
| To team | Platform L1 |

#### Path 2 · on **Platform NOC** (optional bypass)

| Field | Value |
|-------|-------|
| To team | Platform L2 |

#### Path 3 · on **Platform L1**

| Field | Value |
|-------|-------|
| To team | Platform L2 |

#### Path 4 · on **Platform L2**

| Field | Value |
|-------|-------|
| To team | Platform L3 |

NOC incidents show Escalate to L1 and Escalate to L2. L1 sees Escalate to L2.
L2 sees Hand off to L3. You cannot add NOC → L3; that pair is not allowed. Route monitoring
to NOC with a [routing rule](routing.md#ex3), then let people walk the chain.

<a id="ex3"></a>

## 06 · Example 3 — Data L2 may use Platform L3

Data has no L3 of its own. Shared specialists live on Platform L3.

#### Add path · on team **Data L2**

| Field | Value |
|-------|-------|
| Allow team from another workspace | On |
| To team | Platform L3 (Platform) |

Without the checkbox, Platform L3 does not appear in the select. You can still
also add Data L3 later; both targets then show in the handoff picker. Incoming paths on
Platform L3 will list Data L2.

<a id="empty"></a>

## 07 · What on-call sees when this is wrong

| On the incident | Fix on Teams |
| --- | --- |
| No Hand off / Escalate button | Owning team has no outgoing path. Open that team, **Add path**. |
| Helper: no escalation targets configured | Same — path missing or target team lost its tier. |
| Handoff fails: nobody on call on the target | Path is fine. Open **Shifts** for the *target team and create a schedule.* |
| Wrong L3 in the picker (Platform L3 on a Data incident) | You added a cross-workspace path. Remove it if that was not intended. |

> **Checkpoint**
>
> Source team outgoing list has the target. Target team incoming list has the source. Open a
> test incident on L2 and confirm **Hand off to L3** lists that L3 team.

---

Routing (who owns a new alert): [routing rules](routing.md).
Day-to-day buttons: [work an incident](incidents.md).
