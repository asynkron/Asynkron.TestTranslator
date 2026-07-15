# JaCoCo XML report fixtures (real, third-party)

These are **unmodified** JaCoCo code-coverage XML reports committed as test
fixtures/examples in public GitHub repositories. They are used **read-only** as
proving material for a format converter. Contents are byte-for-byte as fetched
from `raw.githubusercontent.com`. No file here was authored or altered by this
project.

All source repositories are under permissive OSI licenses (Apache-2.0 / MIT).

| local file | root | format |
|---|---|---|
| `JaCoCo0.7.7.xml` | `<report>` DTD 1.0 | JaCoCo 0.7.7 output |
| `JaCoCo0.8.3.xml` | `<report>` DTD 1.1 | JaCoCo 0.8.3 output |
| `sample_vokal.xml` | `<report>` DTD 1.0 | small single-package Android report |
| `jacoco_reporter_test.xml` | `<report>` DTD 1.1 | larger 2-package report |

---

## JaCoCo0.7.7.xml
- Source repo: danielpalme/ReportGenerator — https://github.com/danielpalme/ReportGenerator
- Raw URL (pinned): https://raw.githubusercontent.com/danielpalme/ReportGenerator/78d0d61089300ec2ced1dad29b120d7d737d946c/src/Testprojects/Java/Reports/JaCoCo0.7.7.xml
- Original path: `src/Testprojects/Java/Reports/JaCoCo0.7.7.xml`
- License: Apache-2.0
- Description: JaCoCo 0.7.7 XML coverage report ("Example Project"), 2 packages / 7 source files, uses the DTD 1.0 `<report>` doctype; includes nested classes and INSTRUCTION/LINE/COMPLEXITY/METHOD/CLASS counters.

## JaCoCo0.8.3.xml
- Source repo: danielpalme/ReportGenerator — https://github.com/danielpalme/ReportGenerator
- Raw URL (pinned): https://raw.githubusercontent.com/danielpalme/ReportGenerator/78d0d61089300ec2ced1dad29b120d7d737d946c/src/Testprojects/Java/Reports/JaCoCo0.8.3.xml
- Original path: `src/Testprojects/Java/Reports/JaCoCo0.8.3.xml`
- License: Apache-2.0
- Description: JaCoCo 0.8.3 XML coverage report ("Example Project"), 2 packages / 7 source files, DTD 1.1 doctype; includes BRANCH counters (mb/cb) and report-level aggregate counters.

## sample_vokal.xml
- Source repo: vokal/jacoco-parse — https://github.com/vokal/jacoco-parse
- Raw URL (pinned): https://raw.githubusercontent.com/vokal/jacoco-parse/7acdc7261e38eb6699e2a012b8495476e4870663/test/assets/sample.xml
- Original path: `test/assets/sample.xml`
- License: MIT
- Description: Small real JaCoCo report ("debug") from an Android app (com/wmbest/myapplicationtest), 1 package / 1 source file; parser test fixture. DTD 1.0 doctype with sessioninfo.

## jacoco_reporter_test.xml
- Source repo: PavanMudigonda/jacoco-reporter — https://github.com/PavanMudigonda/jacoco-reporter
- Raw URL (pinned): https://raw.githubusercontent.com/PavanMudigonda/jacoco-reporter/112997f0c32da82d8bfa4a972b4afe67a15529fe/jacoco-report/test.xml
- Original path: `jacoco-report/test.xml`
- License: MIT
- Description: Larger JaCoCo XML report (report name "Pester (04/16/2021 11:51:47)"), 2 packages / 6 source files, 78 counter elements, DTD 1.1 doctype; used as the action's input test fixture.
