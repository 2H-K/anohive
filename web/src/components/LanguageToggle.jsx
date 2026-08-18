import { useLanguage } from '../i18n';

export default function LanguageToggle() {
  const { locale, switchLanguage } = useLanguage();

  return (
    <button
      onClick={switchLanguage}
      className="lang-toggle"
      title={locale === 'en' ? '切换到中文' : 'Switch to English'}
      aria-label={locale === 'en' ? '切换到中文' : 'Switch to English'}
    >
      <span className="lang-toggle-icon" aria-hidden="true">
        {locale === 'en' ? '中' : 'EN'}
      </span>
      <span className="lang-toggle-text">
        {locale === 'en' ? '中文' : 'English'}
      </span>
    </button>
  );
}
