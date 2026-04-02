# LickLib — SLO Dokumentation

## Inhaltsverzeichnis

- [Begriffe und Hierarchie](#begriffe-und-hierarchie)
- [Histogramme und Quantile](#histogramme-und-quantile)
- [SLIs — Was wir messen](#slis--was-wir-messen)
- [SLOs — Unsere Ziele](#slos--unsere-ziele)
- [Error Budget](#error-budget)
- [Alerting](#alerting)
- [PromQL Referenz](#promql-referenz)

---

## Begriffe und Hierarchie

```
SLA  (Service Level Agreement)
 └── SLO  (Service Level Objective)
      └── SLI  (Service Level Indicator)
           └── Error Budget
```

### SLI — Service Level Indicator

Der SLI ist die **rohe Messung** — eine konkrete Zahl die beschreibt wie gut der Service gerade läuft. In unserem Fall sind das PromQL-Queries gegen Prometheus.

Beispiel: "In den letzten 5 Minuten waren 97% aller Uploads erfolgreich."

### SLO — Service Level Objective

Das SLO ist das **Ziel** das wir uns für einen SLI setzen. Es ist eine interne Vereinbarung im Team — kein Vertrag mit Nutzern.

Beispiel: "Die Upload-Erfolgsrate muss mindestens 99% betragen, gemessen über 30 Tage."

### SLA — Service Level Agreement

Das SLA ist ein **vertragliches Versprechen** gegenüber Nutzern oder Kunden, oft mit finanziellen Konsequenzen bei Verletzung. Das SLA ist typischerweise etwas lockerer als das interne SLO — man verspricht Nutzern z.B. 99%, arbeitet intern aber auf 99.5% hin, damit man einen Puffer hat.

Für LickLib (noch kein Produkt mit paying customers) definieren wir nur SLOs.

### Error Budget

Das Error Budget ist die **erlaubte Fehlerzeit** die sich aus dem SLO ergibt. Wenn das SLO 99% Erfolgsrate über 30 Tage fordert, dann sind 1% Fehler erlaubt — das ist das Error Budget.

Es beantwortet die Frage: "Wie viel darf noch schiefgehen bevor wir das SLO verletzen?"

---

## Histogramme und Quantile

### Was ist ein Histogram?

Wenn du die Latenz von 1000 Requests messen willst, kannst du nicht einfach den Durchschnitt nehmen — der wird von Ausreißern verfälscht. Stattdessen verteilt Prometheus die Messungen in **Buckets** (Eimer).

Für `licklib_track_upload_duration_seconds` sieht das so aus:

```
Bucket le="0.1"   → Anzahl Uploads die unter 100ms dauerten
Bucket le="0.25"  → Anzahl Uploads die unter 250ms dauerten
Bucket le="0.5"   → Anzahl Uploads die unter 500ms dauerten
Bucket le="1"     → Anzahl Uploads die unter 1s dauerten
Bucket le="2.5"   → Anzahl Uploads die unter 2.5s dauerten
Bucket le="+Inf"  → alle Uploads (Gesamtanzahl)
```

`le` steht für "less than or equal". Die Buckets sind kumulativ — wenn 8 Uploads unter 250ms dauerten, dann sind automatisch auch alle Uploads unter 500ms mindestens 8.

**Konkretes Beispiel aus deinen Metriken:**

```
licklib_track_upload_duration_seconds_bucket{le="0.1"}  = 1
licklib_track_upload_duration_seconds_bucket{le="0.25"} = 1
licklib_track_upload_duration_seconds_bucket{le="+Inf"} = 1
licklib_track_upload_duration_seconds_sum               = 0.083
```

Das bedeutet: 1 Upload, dauerte 83ms, fiel in den Bucket "unter 100ms".

### Was ist ein Quantil / Perzentil?

Ein **p95-Quantil** (95. Perzentil) beantwortet die Frage:

> "Unter welchem Wert liegen 95% aller Messungen?"

Wenn p95 der Upload-Latenz 2 Sekunden ist, dann dauerten 95% aller Uploads weniger als 2 Sekunden — und 5% dauerten länger.

**Warum nicht einfach den Durchschnitt nehmen?**

Stell dir vor du hast 100 Requests:
- 99 Requests dauern 10ms
- 1 Request dauert 10.000ms (10 Sekunden — Timeout)

```
Durchschnitt: (99 × 10 + 1 × 10000) / 100 = 109ms
p95:          10ms   ← 95% der Nutzer warten nur 10ms
p99:          10000ms ← das 1% Problem wird sichtbar
```

Der Durchschnitt sagt "alles gut", aber p99 zeigt dir dass 1% der Nutzer 10 Sekunden warten. **Quantile lügen nicht.**

### PromQL: histogram_quantile

```promql
histogram_quantile(0.95, rate(licklib_track_upload_duration_seconds_bucket[5m]))
```

Breakdown:

```
histogram_quantile(
  0.95,                                              -- p95, also 95. Perzentil
  rate(                                              -- Rate der Zunahme pro Sekunde
    licklib_track_upload_duration_seconds_bucket     -- die Bucket-Zeitreihe
    [5m]                                             -- über die letzten 5 Minuten
  )
)
```

`rate()` ist nötig weil die Buckets Counters sind (zählen immer hoch). `rate()` berechnet die Zunahme pro Sekunde im Zeitfenster — damit bekommt `histogram_quantile` eine Verteilung der aktuellen Periode statt aller Zeiten.

**Warum `_bucket` suffix?**

Prometheus erstellt aus einem Histogram automatisch drei Zeitreihen:
```
licklib_track_upload_duration_seconds_bucket  ← die Buckets (für histogram_quantile)
licklib_track_upload_duration_seconds_sum     ← Summe aller Werte (für Durchschnitt)
licklib_track_upload_duration_seconds_count   ← Anzahl Messungen
```

`histogram_quantile` braucht explizit die `_bucket`-Reihe.

**Durchschnitt berechnen (zum Vergleich):**
```promql
rate(licklib_track_upload_duration_seconds_sum[5m])
/
rate(licklib_track_upload_duration_seconds_count[5m])
```

---

## SLIs — Was wir messen

### SLI 1 — Upload-Erfolgsrate

**Was:** Anteil erfolgreicher Track-Uploads an allen Uploads.

**Warum:** Der Upload ist die Kernfunktion von LickLib. Wenn Uploads fehlschlagen verliert der Nutzer seinen Content — kritischster SLI.

**Metrik:** `licklib_track_uploads_total` mit Label `status` ("success" / "failed")

```promql
-- Erfolgsrate in Prozent (letzten 5 Minuten)
rate(licklib_track_uploads_total{status="success"}[5m])
/
rate(licklib_track_uploads_total[5m])
* 100
```

**Einheit:** Prozent (%)

---

### SLI 2 — Upload-Latenz p95

**Was:** 95% aller Uploads müssen unter diesem Wert abgeschlossen sein.

**Warum:** Nutzer warten aktiv auf den Upload-Abschluss. Zu lange Wartezeiten führen zu Abbrüchen.

**Metrik:** `licklib_track_upload_duration_seconds` (Histogram)

```promql
-- p95 Upload-Dauer in Sekunden
histogram_quantile(
  0.95,
  rate(licklib_track_upload_duration_seconds_bucket[5m])
)
```

**Einheit:** Sekunden

---

### SLI 3 — HTTP-Fehlerrate (5xx)

**Was:** Anteil der Server-Fehler (5xx) an allen HTTP-Requests.

**Warum:** 5xx-Fehler sind Fehler auf unserer Seite — der Server hat versagt, nicht der Client. 4xx-Fehler (401, 403, 404) sind Nutzerfehler und zählen nicht zur Fehlerrate.

**Metrik:** `licklib_http_requests_total` mit Label `status`

```promql
-- 5xx Fehlerrate in Prozent
rate(licklib_http_requests_total{status=~"5.."}[5m])
/
rate(licklib_http_requests_total[5m])
* 100
```

`status=~"5.."` ist ein Regex-Match — alle Status-Codes die mit 5 beginnen.

**Einheit:** Prozent (%)

---

### SLI 4 — API-Latenz p95

**Was:** 95% aller HTTP-Requests müssen unter diesem Wert beantwortet werden.

**Warum:** Allgemeine Responsiveness der API. Träge APIs verlieren Nutzer.

**Metrik:** `licklib_http_request_duration_seconds` (HistogramVec mit Labels method, path, status)

```promql
-- p95 API-Latenz in Sekunden (alle Endpoints)
histogram_quantile(
  0.95,
  rate(licklib_http_request_duration_seconds_bucket[5m])
)

-- p95 nur für einen bestimmten Endpoint
histogram_quantile(
  0.95,
  rate(licklib_http_request_duration_seconds_bucket{path="/tracks",method="POST"}[5m])
)
```

**Einheit:** Sekunden

---

## SLOs — Unsere Ziele

| # | Name | SLI | Ziel | Zeitfenster |
|---|------|-----|------|-------------|
| 1 | Upload-Erfolgsrate | SLI 1 | ≥ 99% | 30 Tage |
| 2 | Upload-Latenz p95 | SLI 2 | ≤ 5s | 30 Tage |
| 3 | HTTP-Fehlerrate | SLI 3 | ≤ 1% | 30 Tage |
| 4 | API-Latenz p95 | SLI 4 | ≤ 500ms | 30 Tage |

### Begründung der Grenzwerte

**SLO 1 — 99% Upload-Erfolgsrate:**
99% bedeutet 1% erlaubte Fehler. Bei 100 Uploads täglich = 1 fehlgeschlagener Upload pro Tag im Durchschnitt. Strenger als 99% wäre für ein frühes Produkt unrealistisch — zu viele externe Abhängigkeiten (MinIO, DB, Netzwerk).

**SLO 2 — 5s Upload-Latenz p95:**
Audiodateien bis 100MB. Bei durchschnittlicher Internetverbindung (10 MB/s Upload) dauert ein 50MB File ~5 Sekunden rein netzwerkseitig. Der Wert ist bewusst großzügig gewählt.

**SLO 3 — 1% HTTP-Fehlerrate:**
1% Server-Fehler ist industrieller Standard für frühe Produkte. Google/Amazon arbeiten auf <0.1%, aber die haben auch deutlich mehr Redundanz.

**SLO 4 — 500ms API-Latenz p95:**
Für reine Lese-Operationen (GET /tracks, GET /users) ist 500ms großzügig. Der Login-Endpoint (Keycloak-Roundtrip) dauert ~270ms alleine, daher kein strengerer Wert für alle Endpoints zusammen.

---

## Error Budget

### Berechnung

```
Error Budget = (1 - SLO) × Zeitfenster in Minuten

Beispiel SLO 1 (99% über 30 Tage):
30 Tage × 24 Stunden × 60 Minuten = 43.200 Minuten
Error Budget = 1% × 43.200 = 432 Minuten = 7 Stunden 12 Minuten
```

### Error Budget Tabelle

| SLO | Ziel | Erlaubte Fehler | Error Budget (30 Tage) |
|-----|------|-----------------|------------------------|
| Upload-Erfolgsrate | 99% | 1% | 432 Min (~7.2h) |
| Upload-Latenz p95 | ≤ 5s | 5% over budget | 2.160 Min (~36h) |
| HTTP-Fehlerrate | 99% | 1% | 432 Min (~7.2h) |
| API-Latenz p95 | ≤ 500ms | 5% over budget | 2.160 Min (~36h) |

### Burn Rate

Die **Burn Rate** beschreibt wie schnell das Error Budget verbraucht wird, relativ zur erlaubten Rate.

```
Burn Rate 1  = Budget wird genau im erlaubten Tempo verbraucht
Burn Rate 2  = Budget wird doppelt so schnell verbraucht
Burn Rate 14 = Budget wird 14x so schnell verbraucht
               → bei dieser Rate ist Budget in ~2 Tagen weg statt 30
```

**Burn Rate Alert (empfohlen):**

```promql
-- Burn Rate für Upload-Erfolgsrate
-- Alert wenn Budget 14x schneller verbraucht wird als erlaubt
(
  rate(licklib_track_uploads_total{status="failed"}[1h])
  /
  rate(licklib_track_uploads_total[1h])
)
/
0.01  -- erlaubte Fehlerrate (1%)
> 14  -- Burn Rate Schwellwert
```

Warum 14? Bei Burn Rate 14 ist das 30-Tage-Budget in ~2 Tagen aufgebraucht — das ist dringend genug für einen Alert, aber nicht so sensitiv dass jeder kurze Fehler auslöst.

---

## Alerting

### Alert-Typen

**1. Threshold Alert** — einfach, direkt, kann "flappen" (schnell an/aus schalten):

```promql
-- Feuert wenn Fehlerrate in letzten 5min über 10%
rate(licklib_track_uploads_total{status="failed"}[5m])
/
rate(licklib_track_uploads_total[5m])
> 0.10
```

**2. Burn Rate Alert** — robuster, berücksichtigt Tempo des Budget-Verbrauchs:

```promql
-- Feuert wenn Error Budget 14x schneller verbraucht wird als erlaubt
(
  rate(licklib_track_uploads_total{status="failed"}[1h])
  /
  rate(licklib_track_uploads_total[1h])
) / 0.01 > 14
```

### Alert Rules für LickLib

#### Alert 1 — Upload-Fehlerrate kritisch

```yaml
# in .extras/prometheus/alerts.yml
groups:
  - name: licklib_slo
    rules:
      - alert: UploadErrorRateHigh
        expr: |
          rate(licklib_track_uploads_total{status="failed"}[5m])
          /
          rate(licklib_track_uploads_total[5m])
          > 0.10
        for: 2m
        labels:
          severity: critical
          slo: upload_success_rate
        annotations:
          summary: "Upload-Fehlerrate zu hoch"
          description: "Fehlerrate ist {{ $value | humanizePercentage }} — SLO erlaubt max 1%"
```

#### Alert 2 — Upload-Latenz zu hoch

```yaml
      - alert: UploadLatencyHigh
        expr: |
          histogram_quantile(
            0.95,
            rate(licklib_track_upload_duration_seconds_bucket[5m])
          ) > 5
        for: 5m
        labels:
          severity: warning
          slo: upload_latency
        annotations:
          summary: "Upload-Latenz p95 überschreitet 5s"
          description: "p95 Latenz ist {{ $value }}s — SLO erlaubt max 5s"
```

#### Alert 3 — HTTP Server-Fehler

```yaml
      - alert: HTTPErrorRateHigh
        expr: |
          rate(licklib_http_requests_total{status=~"5.."}[5m])
          /
          rate(licklib_http_requests_total[5m])
          > 0.01
        for: 2m
        labels:
          severity: critical
          slo: http_error_rate
        annotations:
          summary: "HTTP 5xx Fehlerrate zu hoch"
          description: "Server-Fehlerrate ist {{ $value | humanizePercentage }}"
```

#### Alert 4 — API-Latenz zu hoch

```yaml
      - alert: APILatencyHigh
        expr: |
          histogram_quantile(
            0.95,
            rate(licklib_http_request_duration_seconds_bucket[5m])
          ) > 0.5
        for: 5m
        labels:
          severity: warning
          slo: api_latency
        annotations:
          summary: "API-Latenz p95 überschreitet 500ms"
          description: "p95 Latenz ist {{ $value }}s auf {{ $labels.path }}"
```

### `for` — Warum Alerts nicht sofort feuern sollten

```yaml
for: 2m
```

Der Alert feuert erst wenn die Bedingung **2 Minuten lang ununterbrochen** wahr ist. Ohne `for` würde jeder kurze Spike (ein fehlgeschlagener Upload, kurzes Netzwerk-Hickup) sofort einen Alert auslösen — das nennt man **Flapping** und führt zu Alert-Fatigue (man ignoriert Alerts weil sie zu oft kommen).

Faustregel:
- `critical` Alerts: `for: 2m` — schnell reagieren
- `warning` Alerts: `for: 5m` — etwas mehr Toleranz

### Severity Levels

| Level | Bedeutung | Reaktion |
|-------|-----------|----------|
| `critical` | SLO wird gerade verletzt oder Error Budget verbrennt schnell | Sofort handeln |
| `warning` | SLO könnte bald verletzt werden | Innerhalb von Stunden handeln |
| `info` | Auffälligkeit, kein SLO-Risiko | Beim nächsten Review ansehen |

### alerts.yml in prometheus.yml einbinden

```yaml
# .extras/prometheus/prometheus.yml
global:
  scrape_interval: 15s

rule_files:
  - "alerts.yml"  # Alert Rules laden

scrape_configs:
  - job_name: 'licklib'
    static_configs:
      - targets: ['host.docker.internal:8080']
```

---

## PromQL Referenz

### Häufig verwendete Funktionen

| Funktion | Bedeutung | Beispiel |
|----------|-----------|---------|
| `rate(m[5m])` | Zunahme pro Sekunde über 5min | `rate(uploads_total[5m])` |
| `increase(m[1h])` | Absolute Zunahme über 1h | `increase(uploads_total[1h])` |
| `histogram_quantile(q, m)` | q-Quantil aus Histogram | `histogram_quantile(0.95, ...)` |
| `sum(m)` | Alle Label-Kombinationen addieren | `sum(rate(requests[5m]))` |
| `by (label)` | Gruppierung | `sum by (status)(rate(...))` |
| `without (label)` | Alle Labels außer diesem | `sum without (instance)(rate(...))` |

### Label-Matching

```promql
{status="200"}      -- exakter Match
{status!="200"}     -- nicht gleich
{status=~"2.."}     -- Regex Match (beginnt mit 2)
{status!~"2.."}     -- Regex nicht Match
```

### Zeitfenster-Größen

```promql
[1m]   -- 1 Minute   (sehr kurzfristig, reagiert schnell aber unruhig)
[5m]   -- 5 Minuten  (Standard für Dashboards)
[1h]   -- 1 Stunde   (für Burn Rate Alerts)
[1d]   -- 1 Tag      (für SLO-Compliance über Zeit)
[30d]  -- 30 Tage    (für monatliche SLO-Reports)
```

Größere Zeitfenster glätten kurzfristige Spikes — kleinere Zeitfenster reagieren schneller aber sind rauschiger. Für Alerts lieber größere Fenster + `for` kombinieren.