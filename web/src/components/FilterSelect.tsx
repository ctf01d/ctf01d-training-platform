import { useState, useRef, useEffect } from "react";
import { useI18n } from "../i18n/I18nContext";

export interface FilterSelectOption {
  id: number;
  /** Text shown for the option and matched against the query. */
  label: string;
  /** Extra text matched against the query but not displayed (e.g. description). */
  search?: string;
}

/**
 * A combobox: a text input that filters a list of options as the user types,
 * instead of dropping the whole list at once. Matches against `label` and the
 * optional `search` text. Used for university and service pickers.
 */
export function FilterSelect({
  options,
  value,
  onChange,
  placeholder = "Search…",
  allowClear = false,
  required = false,
}: {
  options: FilterSelectOption[];
  value: number | null;
  onChange: (id: number | null) => void;
  placeholder?: string;
  allowClear?: boolean;
  required?: boolean;
}) {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const selected = options.find((o) => o.id === value) ?? null;

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node))
        setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const q = query.trim().toLowerCase();
  const filtered = (
    q
      ? options.filter((o) =>
          `${o.label} ${o.search ?? ""}`.toLowerCase().includes(q),
        )
      : options
  ).slice(0, 50);

  return (
    <div className="filter-select" ref={ref}>
      <input
        ref={inputRef}
        type="text"
        value={open ? query : (selected?.label ?? "")}
        placeholder={selected ? selected.label : t(placeholder)}
        onFocus={() => {
          setOpen(true);
          setQuery("");
        }}
        onChange={(e) => {
          setQuery(e.target.value);
          setOpen(true);
        }}
      />
      {required && (
        // Mirrors the selected id so HTML form validation tracks the actual
        // selection rather than whatever free text is in the visible input.
        // Kept focusable (not display:none) so the browser can surface its
        // validation message; focus is redirected to the real input.
        <input
          className="filter-select-validity"
          tabIndex={-1}
          aria-hidden="true"
          required
          value={value ?? ""}
          onChange={() => {}}
          onFocus={() => inputRef.current?.focus()}
        />
      )}
      {allowClear && selected && (
        <button
          type="button"
          className="filter-select-clear"
          title={t("Clear")}
          onClick={() => {
            onChange(null);
            setQuery("");
          }}
        >
          ×
        </button>
      )}
      {open && (
        <ul className="filter-select-menu">
          {filtered.length === 0 ? (
            <li className="filter-select-empty">{t("No matches")}</li>
          ) : (
            filtered.map((o) => (
              <li key={o.id}>
                <button
                  type="button"
                  className={o.id === value ? "is-selected" : undefined}
                  onClick={() => {
                    onChange(o.id);
                    setOpen(false);
                    setQuery("");
                  }}
                >
                  {o.label}
                </button>
              </li>
            ))
          )}
        </ul>
      )}
    </div>
  );
}
