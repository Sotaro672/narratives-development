// frontend/src/components/layout/header/HeaderSettingsPanel.tsx
import SettingsMenu from "../SettingsMenu";

type HeaderSettingsPanelProps = {
  settingsOpen: boolean;
  closeSettings: () => void;
};

export default function HeaderSettingsPanel({
  settingsOpen,
  closeSettings,
}: HeaderSettingsPanelProps) {
  return (
    <>
      <button
        type="button"
        className={`header__menu-backdrop ${
          settingsOpen ? "header__menu-backdrop--open" : ""
        }`}
        onClick={closeSettings}
        aria-label="設定を閉じる"
        aria-hidden={!settingsOpen}
        tabIndex={settingsOpen ? 0 : -1}
      />

      <aside
        className={`header__settings-panel ${
          settingsOpen ? "header__settings-panel--open" : ""
        }`}
        aria-hidden={!settingsOpen}
      >
        <div className="header__settings-panel-inner settings-page settings-page--sidebar">
          <SettingsMenu onItemClick={closeSettings} />
        </div>
      </aside>
    </>
  );
}