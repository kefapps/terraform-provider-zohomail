# AGENTS.md

## Repo Guardrails

- Scope: rester sur le provider Terraform Zoho Mail standalone. Ne pas rebasculer vers Ansyo ou un provider Zoho générique.
- Docs: lire `README.md` en premier, puis seulement le fichier pertinent.
- Provider work: garder une logique open-source classique compatible Terraform Registry.
- Quality gate: avant tout push, `make quality` doit passer localement. Utiliser `make sonar-local` pour un scan de debug sur worktree sale; réserver `make quality` à la certification stricte sur worktree propre.
- Docs generation: relancer `make generate` après tout changement de schéma, d’exemple ou de doc template.

Details: `README.md`
