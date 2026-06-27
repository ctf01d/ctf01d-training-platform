import {
  createContext,
  useContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useAuth } from "../auth/AuthContext";
import {
  detectPreferredLanguage,
  languageLiteral,
  normalizeLanguage,
  roleLiteral,
  setCurrentLanguage,
  translateLiteral,
  type Language,
  type TranslationParams,
} from "./runtime";

interface I18nContextValue {
  language: Language;
  t: (source: string, params?: TranslationParams) => string;
  roleLabel: (role?: string | null) => string;
  languageLabel: (language?: string | null) => string;
  setLanguage: (language: Language) => void;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const [language, setLanguage] = useState<Language>(() => {
    const initialLanguage = detectPreferredLanguage();
    setCurrentLanguage(initialLanguage);
    return initialLanguage;
  });

  const applyLanguage = useCallback((value?: string | null) => {
    const nextLanguage = normalizeLanguage(value);
    setCurrentLanguage(nextLanguage);
    setLanguage((currentLanguage) =>
      currentLanguage === nextLanguage ? currentLanguage : nextLanguage,
    );
  }, []);

  useEffect(() => {
    if (!user?.language) return;
    applyLanguage(user.language);
  }, [applyLanguage, user?.language]);

  const value = useMemo<I18nContextValue>(
    () => ({
      language,
      t: (source, params) => translateLiteral(language, source, params),
      roleLabel: (role) => translateLiteral(language, roleLiteral(role)),
      languageLabel: (value) =>
        translateLiteral(language, languageLiteral(value)),
      setLanguage: applyLanguage,
    }),
    [applyLanguage, language],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
