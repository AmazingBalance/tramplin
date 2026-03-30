# Presentation

Исходник презентации находится в [slides.md](/home/evgen/Desktop/Tramplin/presentation/slides.md).

Структура:

- `slides.md` — основной Marp deck
- `assets/roles.svg` — роли и пользовательские потоки
- `assets/architecture.svg` — архитектура решения
- `assets/data-model.svg` — укрупнённая схема данных
- `assets/ui-structure.svg` — структура интерфейса

Экспорт в PDF через Marp CLI:

```bash
npx @marp-team/marp-cli slides.md --allow-local-files --pdf -o tramplin-presentation.pdf
```

Альтернатива:

- открыть `slides.md` в VS Code с расширением Marp for VS Code
- выполнить `Marp: Export slide deck...`
