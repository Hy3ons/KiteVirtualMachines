# Public policy drafts

`/privacy` and `/terms` are public, session-independent policy drafts. They assume
Google sign-in will be added; this change does not implement OAuth, consent
recording, account deletion, log rotation or backup deletion. The pages explicitly
distinguish proposed Google behavior from the existing email/password flow.

The footer links to both documents. Neither document has an effective date or
claims that visiting the website constitutes acceptance.

Before adopting the policies, the actual instance operator must finalize:

- Legal operator name, privacy contact/responsible department and contact details.
- Actual Google scopes, returned claims, account linking/deletion and revocation.
- Actual request/security logging fields, retention and purge schedule.
- VM data, credentials and backup lifecycle, including account/resource deletion.
- Hosting, mail, backups, CDNs and other processors, third-party disclosures and
  any international transfers, with the actual entities/countries and legal basis.
- Effective date, versioning, notices and any required consent collection.

No operator identity, email, storage country, retention period or paid service
terms are invented. The existing admin-contact setting is authenticated and
free-form; it is not exposed as an anonymous legal contact endpoint.

## Sources checked on 2026-10-30

- [Personal Information Protection Act, Article 30](https://www.law.go.kr/LSW/lsLinkCommonInfo.do?lsJoLnkSeq=1029334833)
  for privacy notice subject areas, including purposes, retention, processors,
  rights and privacy contact information.
- [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect)
  for authentication identifiers and email/profile scope distinctions.
- [Google API Services User Data Policy](https://developers.google.com/terms/api-services-user-data-policy)
  for transparent disclosures, purpose limitation and minimum permissions.
- [Google Privacy Policy](https://policies.google.com/privacy)
  for Google's own processing, separately linked from Kite's draft.

QA follows the repository contract: types, lint, production build and anonymous
HTTP route/link checks. Visual and interaction QA belongs to the user.
