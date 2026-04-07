import React, { useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useNavigate, useLocation } from 'react-router-dom';
import { Plus, Box, Search, ChevronDown, LogOut, Shield } from 'lucide-react';
import { useEffect, useRef } from 'react';
import JobsPage from './pages/JobsPage';
import JobDetailsPage from './pages/JobDetailsPage';
import CreateJobModal from './components/CreateJobModal';
import ProtectedRoute from './components/ProtectedRoute';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuth } from 'react-oidc-context';

import { LanguageProvider, useTranslation } from './context/LanguageContext';
import { SearchProvider, useSearch } from './context/SearchContext';

const queryClient = new QueryClient();

const LanguageSwitcher = () => {
  const { language, setLanguage } = useTranslation();
  const languages = [
    { code: 'ru', label: 'Русский' },
    { code: 'en', label: 'English' }
  ];

  return (
    <div className="lang-switcher">
      {languages.map((lang) => (
        <button
          key={lang.code}
          onClick={() => setLanguage(lang.code as any)}
          className={`lang-btn ${language === lang.code ? 'active' : ''}`}
        >
          {lang.label}
        </button>
      ))}
    </div>
  );
};

const getInitials = (user: any) => {
  if (!user?.profile) return '?';

  const givenName = user.profile.given_name;
  const familyName = user.profile.family_name;
  const name = user.profile.name;
  const preferredUsername = user.profile.preferred_username;

  if (givenName && familyName) {
    return (givenName[0] + (familyName[0] || '')).toUpperCase();
  }

  if (name) {
    const parts = name.split(' ');
    if (parts.length >= 2) {
      return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
    }
    return parts[0][0].toUpperCase();
  }

  if (preferredUsername) {
    return preferredUsername.slice(0, 2).toUpperCase();
  }

  return '?';
};

const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isModalOpen, setModalOpen] = useState(false);
  const { t } = useTranslation();
  const { searchQuery, setSearchQuery } = useSearch();
  const navigate = useNavigate();
  const location = useLocation();
  const auth = useAuth();
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsMenuOpen(false);
      }
    };

    if (isMenuOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isMenuOpen]);

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSearchQuery(value);

    // Navigate home if we're not already there and there's a search query
    if (location.pathname !== '/' && value.trim() !== '') {
      navigate('/');
    }
  };

  return (
    <div className="min-h-screen bg-transparent text-white flex flex-col">
      {/* Header */}
      <header className="main-header">
        <Link to="/" className="nav-logo">
          <div className="logo-icon pulsing">
            <Box size={24} />
          </div>
          <span className="text-xl font-bold tracking-tight">
            SwarmRender
          </span>
        </Link>

        <div className="flex items-center gap-4">
          <LanguageSwitcher />

          <div className="search-container">
            <Search className="search-icon" size={16} />
            <input
              type="text"
              placeholder={t('common.search_placeholder')}
              className="search-input"
              value={searchQuery}
              onChange={handleSearchChange}
            />
          </div>
          <div className="flex items-center gap-3">
            {auth.isAuthenticated ? (
              <div className="relative" ref={menuRef}>
                <button
                  onClick={() => setIsMenuOpen(!isMenuOpen)}
                  className="user-menu-btn flex items-center gap-2 hover:bg-white/10 p-1 pr-2 rounded-full transition-all"
                >
                  <div className="user-avatar">
                    {getInitials(auth.user)}
                  </div>
                  <ChevronDown size={14} className={`text-white/40 transition-transform ${isMenuOpen ? 'rotate-180' : ''}`} />
                </button>

                {isMenuOpen && (
                  <div className="user-dropdown animate-in fade-in slide-in-from-top-2 duration-200">
                    <div className="p-4">
                      <div className="flex items-center gap-3">
                        <div className="user-avatar-large">
                          {getInitials(auth.user)}
                        </div>
                        <div className="flex flex-col">
                          <span className="text-sm font-bold truncate max-w-[150px]">
                            {auth.user?.profile.name || auth.user?.profile.preferred_username}
                          </span>
                        </div>
                      </div>
                    </div>

                    <div className="p-2">
                      <button
                        onClick={() => auth.signoutRedirect()}
                        className="dropdown-item text-red-400 hover:bg-red-400/10"
                      >
                        <LogOut size={16} />
                        <span>Выйти</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <button
                onClick={() => auth.signinRedirect()}
                className="btn-secondary"
              >
                Login
              </button>
            )}
          </div>

          <div className="w-[1px] h-6 bg-white/10 mx-1"></div>

          <button
            onClick={() => setModalOpen(true)}
            className="btn-primary"
            disabled={!auth.isAuthenticated}
          >
            <Plus size={18} /> {t('common.add_project')}
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 p-6 max-w-7xl mx-auto">
        {children}
      </main>

      <CreateJobModal isOpen={isModalOpen} onClose={() => setModalOpen(false)} />
    </div>
  );
};

function App() {
  return (
    <LanguageProvider>
      <SearchProvider>
        <QueryClientProvider client={queryClient}>
          <Router>
            <Layout>
              <Routes>
                <Route path="/" element={<ProtectedRoute><JobsPage /></ProtectedRoute>} />
                <Route path="/job/:id" element={<ProtectedRoute><JobDetailsPage /></ProtectedRoute>} />
              </Routes>
            </Layout>
          </Router>
        </QueryClientProvider>
      </SearchProvider>
    </LanguageProvider>
  );
}

export default App;
