import React, { useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useNavigate, useLocation } from 'react-router-dom';
import { Plus, Box, Search } from 'lucide-react';
import JobsPage from './pages/JobsPage';
import JobDetailsPage from './pages/JobDetailsPage';
import CreateJobModal from './components/CreateJobModal';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

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

const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isModalOpen, setModalOpen] = useState(false);
  const { t } = useTranslation();
  const { searchQuery, setSearchQuery } = useSearch();
  const navigate = useNavigate();
  const location = useLocation();

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
            RenderCore
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
          <button 
            onClick={() => setModalOpen(true)}
            className="btn-primary"
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
              <Route path="/" element={<JobsPage />} />
              <Route path="/job/:id" element={<JobDetailsPage />} />
            </Routes>
          </Layout>
        </Router>
        </QueryClientProvider>
      </SearchProvider>
    </LanguageProvider>
  );
}

export default App;
