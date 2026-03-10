# Changelog
All notable changes to this project will be documented in this file. See [conventional commits](https://www.conventionalcommits.org/) for commit guidelines.

- - -
## [v2.1.0](https://github.com/strubio-ray/marvin-time-tracker/compare/23eb669e73e023648ef27e41aed839a84065f5f9..v2.1.0) - 2026-03-10
#### Features
- (**build**) add bump-userscript recipe for semver version bumps - ([6c5fbc2](https://github.com/strubio-ray/marvin-time-tracker/commit/6c5fbc2d5efdc4903febfb3ba9d2dfe3003ade4e)) - Steven
- (**userscript**) add API key authentication and fetch-based SSE - ([23eb669](https://github.com/strubio-ray/marvin-time-tracker/commit/23eb669e73e023648ef27e41aed839a84065f5f9)) - Steven

- - -

## [v2.0.0](https://github.com/strubio-ray/marvin-time-tracker/compare/70d2645feda72bb7eec4e1588844f5b46c134a35..v2.0.0) - 2026-03-10
#### Features
- (**ios**) add test target with Swift Testing framework - ([3fa45a2](https://github.com/strubio-ray/marvin-time-tracker/commit/3fa45a297c9bf2e4cfa9b2ccddc9cddbcf0bb07a)) - Steven
- add 30-second TTL cache to GET /tasks - ([69d2f40](https://github.com/strubio-ray/marvin-time-tracker/commit/69d2f408ee2b0f88ddab75db7c4071ffbc7a226d)) - Steven
- add per-IP rate limiting on webhook endpoints - ([d994190](https://github.com/strubio-ray/marvin-time-tracker/commit/d9941907841e2b6461d9eb38285b5733a177a63c)) - Steven
- add graceful shutdown with signal handling - ([96f7f31](https://github.com/strubio-ray/marvin-time-tracker/commit/96f7f31ec0b36fdd192a984997cd06100088399a)) - Steven
- ![BREAKING](https://img.shields.io/badge/BREAKING-red) replace sideshow/apns2 with stdlib APNs HTTP/2 client using jwt/v5 - ([eb4b0cf](https://github.com/strubio-ray/marvin-time-tracker/commit/eb4b0cff24075c27c37366aa680eb655d7c17f0a)) - Steven
#### Bug Fixes
- (**ios**) replace hardcoded xcodegen path and add explicit error logging - ([a06e504](https://github.com/strubio-ray/marvin-time-tracker/commit/a06e504e885f2b13d691baec8c000f919259d4de)) - Steven
- prevent JSON injection in marvin Track() and use bytes.NewReader - ([30e8f29](https://github.com/strubio-ray/marvin-time-tracker/commit/30e8f291277f26947c93bb41a4a8a8c8f970c0b5)) - Steven
- add request body limits and HTTP server timeouts - ([0598cd5](https://github.com/strubio-ray/marvin-time-tracker/commit/0598cd52d5cdde43852cf1058ed7c6ce7f9fecf6)) - Steven
- add auth to /events and /history endpoints - ([478572f](https://github.com/strubio-ray/marvin-time-tracker/commit/478572f0d4cafd51c51ee31206c4ca6713d3361f)) - Steven
- update dependencies and bump Go version - ([70d2645](https://github.com/strubio-ray/marvin-time-tracker/commit/70d2645feda72bb7eec4e1588844f5b46c134a35)) - Steven
#### Refactoring
- (**ios**) extract shared constants and ElapsedTimerText view - ([2881edd](https://github.com/strubio-ray/marvin-time-tracker/commit/2881edd0f507a54ecef3bdf995b123df8f21dcdc)) - Steven
- (**ios**) introduce DI protocols for TrackingViewModel - ([c888de6](https://github.com/strubio-ray/marvin-time-tracker/commit/c888de680582f6bf602dd334b35818b2b2b223ba)) - Steven
- (**server**) centralize token consumption and atomic state operations - ([4d1cb28](https://github.com/strubio-ray/marvin-time-tracker/commit/4d1cb28ae37337228a7fdccaa341bbf93e16efa9)) - Steven
- define BrokerPublisher and SessionRecorder interfaces - ([1bb95fe](https://github.com/strubio-ray/marvin-time-tracker/commit/1bb95fe94960fa31f946974994d01ab57ae48477)) - Steven
- pass tokens as params to notify functions - ([bb2de5e](https://github.com/strubio-ray/marvin-time-tracker/commit/bb2de5e39f07193e23e7e239ebd6e6cd2347149a)) - Steven
- extract atomicWriteJSON helper and ClearTracking method - ([8001351](https://github.com/strubio-ray/marvin-time-tracker/commit/8001351d10f99b51a3e50e271aced8dcf79acc40)) - Steven
- gate raw webhook body logging behind debug mode - ([95830c5](https://github.com/strubio-ray/marvin-time-tracker/commit/95830c50fa56d1a1294e37fae7637fb3dc2f47a4)) - Steven

- - -

## [v1.5.1](https://github.com/strubio-ray/marvin-time-tracker/compare/3aa75fa5a0fe94dc66e2dc84491716477d52e120..v1.5.1) - 2026-03-10
#### Bug Fixes
- (**ios**) add missing info.path for XcodeGen 2.45 compatibility - ([3aa75fa](https://github.com/strubio-ray/marvin-time-tracker/commit/3aa75fa5a0fe94dc66e2dc84491716477d52e120)) - Steven

- - -

## [v1.5.0](https://github.com/strubio-ray/marvin-time-tracker/compare/58199e077ae79d5d26c3c86025e015b9a67e2138..v1.5.0) - 2026-03-10
#### Features
- add API key auth and tasks endpoint - ([a25606a](https://github.com/strubio-ray/marvin-time-tracker/commit/a25606a81897d1af0bd2af758713225cdc53b62e)) - Steven
#### Documentation
- update release pipeline to use cocogitto - ([f369a4a](https://github.com/strubio-ray/marvin-time-tracker/commit/f369a4af06700c44ceb790981493d35d91dd1ef4)) - Steven
#### Refactoring
- remove polling fallback in favor of webhooks only - ([bf37e69](https://github.com/strubio-ray/marvin-time-tracker/commit/bf37e6937e8262622cf9bbceb9a36f25413b5bbf)) - Steven
- remove bounce-back guard logic - ([58199e0](https://github.com/strubio-ray/marvin-time-tracker/commit/58199e077ae79d5d26c3c86025e015b9a67e2138)) - Steven

- - -

Changelog generated by [cocogitto](https://github.com/cocogitto/cocogitto).