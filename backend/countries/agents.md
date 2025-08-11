# countries package: Static country list

Overview
- Endpoint: GET /countries returns a small static list with fields {id, name, short_code, phone_code}.

Environment variables
- None.

How it works
- Provides a fixed array for development; shape matches the Flutter CountryModel.

Good practices
- Move to a DB table or external ISO source; keep codes normalized.
- Add caching headers; support filtering/search.

Architecture notes
- Standalone utility; registered by main.go.
