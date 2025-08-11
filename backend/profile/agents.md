# profile package: User profile CRUD and media

Overview
- Endpoints: GET /user-detail/:id, POST /user-detail/:id
- Supports JSON updates and multipart/form-data image upload stored under media/user_<id>/profile.jpg

Environment variables
- None directly. Uses filesystem paths relative to the backend root.

How it works
- Auth: requires Bearer token issued by login to allow updates; GET is public in current code.
- GET returns user profile fields from DB.
- POST (multipart): expects `profile_image` file, saves under media/user_<id>/, updates DB path.
- POST (JSON): accepts first_name, last_name, city, profession; updates DB.

Good practices
- Enforce auth for GET as well if profiles are private.
- Validate and limit upload size and allowed mime types.
- Serve /media with proper cache-control and access rules.
- Sanitize user-provided strings; centralize validation.

Architecture notes
- profile depends on login (token decode) and migrations (DB ops).
- Media paths are relative; consider configuring a MEDIA_ROOT via env.
