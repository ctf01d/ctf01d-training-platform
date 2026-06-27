import { THEMES, type ThemeId } from "../theme";
import { useI18n } from "../i18n/I18nContext";

export default function ThemeSection({
  value,
  onChange,
  disabled,
}: {
  value: ThemeId;
  onChange: (id: ThemeId) => void;
  disabled?: boolean;
}) {
  const { t } = useI18n();

  return (
    <div className="detail-section">
      <div className="section-head">
        <h3>{t("Theme")}</h3>
      </div>
      <div className="theme-picker">
        <div className="theme-options" role="group" aria-label={t("Theme")}>
          {THEMES.map((option) => (
            <button
              key={option.id}
              type="button"
              disabled={disabled}
              className={`theme-swatch theme-swatch--${option.id} ${
                value === option.id ? "is-active" : ""
              }`}
              onClick={() => onChange(option.id)}
              aria-pressed={value === option.id}
              title={t(option.label)}
            >
              <span className="theme-swatch-dot" aria-hidden="true" />
              <span className="theme-swatch-label">{t(option.label)}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
