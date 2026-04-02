import React, { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { jobService, Job } from '../services/api';
import { useNavigate } from 'react-router-dom';
import { Clock, HardDrive, Layers, ChevronRight, Play, Download } from 'lucide-react';
import { motion } from 'framer-motion';
import { useTranslation } from '../context/LanguageContext';
import { wsService } from '../services/ws';
import ProgressBar from '../components/common/ProgressBar';
import { useSearch } from '../context/SearchContext';
import PreviewPlaceholder from '../components/common/PreviewPlaceholder';

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const { t } = useTranslation();
  const statusClass = status.toLowerCase() || 'created';

  return (
    <span className={`status-badge status-${statusClass} ${status === 'PROCESSING' ? 'pulsing' : ''}`}>
      {t(`status.${status}`)}
    </span>
  );
};

const JobCard: React.FC<{ job: Job }> = ({ job }) => {
  const navigate = useNavigate();
  const { t } = useTranslation();

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ y: -5 }}
      onClick={() => navigate(`/job/${job.id}`)}
      className="glass glass-interactive group cursor-pointer overflow-hidden job-card"
    >
      {/* Preview Section */}
      <div className="job-card-preview">
        {job.preview_url ? (
          <img src={job.preview_url} alt={job.name} className="w-full h-full object-cover" />
        ) : (
          <PreviewPlaceholder name={job.name} />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-60"></div>
        <div className="absolute top-3 left-3">
          <StatusBadge status={job.status} />
        </div>
      </div>

      {/* Info Section */}
      <div className="py-2">
        <h3 className="text-lg font-bold truncate text-white/90 group-hover:text-cyan-400 transition-colors">
          {job.name}
        </h3>

        <div className="flex items-center gap-4 text-white/40 text-[10px] mt-2">
          <div className="flex items-center gap-2 uppercase font-bold tracking-tight">
            <Layers size={14} /> {job.frame_start}-{job.frame_end}
          </div>
          <div className="flex items-center gap-2 uppercase font-bold tracking-tight">
            <HardDrive size={14} /> {job.worker_count} {t('common.workers')}
          </div>
        </div>

        {job.status === 'PROCESSING' && (
          <div className="mt-5">
            <ProgressBar
              current={job.completed_frames || 0}
              total={job.frame_end - job.frame_start + 1}
              height={4}
            />
          </div>
        )}

        <div className="mt-4 pt-4 border-t border-white/5 flex items-center justify-between">
          <div className="flex items-center gap-2 text-white/40 text-[10px]">
            <Clock size={12} /> {new Date(job.created_at).toLocaleDateString()}
          </div>

          <div className="flex items-center gap-3">
            {job.status === 'COMPLETED' && job.result_path && (
              <a
                href={job.result_path}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="w-8 h-8 bg-emerald-500 rounded-full flex items-center justify-center text-white hover:bg-emerald-400 transition-all hover:scale-110 shadow-lg shadow-emerald-500/20"
                title={t('common.download_result')}
              >
                <Download size={18} />
              </a>
            )}
            <div className="w-8 h-8 bg-cyan-500 rounded-full flex items-center justify-center text-white hover:scale-110 transition-all shadow-lg shadow-cyan-500/20">
              <Play size={18} fill="currentColor" className="ml-0.5" />
            </div>
            <ChevronRight size={20} className="text-white/20 group-hover:text-white group-hover:translate-x-1 transition-all ml-1" />
          </div>
        </div>
      </div>
    </motion.div>
  );
};

const JobsPage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { searchQuery } = useSearch();
  const { data: jobs, isLoading, error } = useQuery({
    queryKey: ['jobs'],
    queryFn: jobService.listJobs,
  });

  // WebSocket for real-time updates
  useEffect(() => {
    const unsubscribe = wsService.subscribe((msg) => {
      if (msg.type === 'job_update' || msg.type === 'job_progress') {
        queryClient.invalidateQueries({ queryKey: ['jobs'] });
      }
    });
    return () => unsubscribe();
  }, [queryClient]);

  if (isLoading) return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
      {[1, 2, 3, 4].map(i => (
        <div key={i} className="glass h-64 animate-pulse bg-white/5 border-white/5"></div>
      ))}
    </div>
  );

  if (error) return (
    <div className="glass p-8 text-center bg-rose-500/5 border-rose-500/20">
      <h2 className="text-xl font-bold text-rose-400">{t('common.error_loading')}</h2>
    </div>
  );

  const filteredJobs = jobs?.filter(job =>
    job.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    job.id.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('jobs_page.title')}</h1>
          <p className="text-white/40 mt-1">{t('jobs_page.subtitle')}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filteredJobs?.map((job) => (
          <JobCard key={job.id} job={job} />
        ))}
      </div>

      {filteredJobs?.length === 0 && (
        <div className="glass p-16 text-center border-dashed border-2 bg-transparent opacity-50">
          <div className="w-16 h-16 bg-white/5 rounded-full flex items-center justify-center mx-auto mb-4">
            <Layers size={32} className="text-white/20" />
          </div>
          <h3 className="text-xl font-semibold">{searchQuery ? t('common.no_results_found') || 'No results found' : t('common.no_active_jobs')}</h3>
          <p className="mt-2 text-sm text-white/40">{searchQuery ? t('common.try_another_search') || 'Try another search query' : t('common.add_first_project')}</p>
        </div>
      )}
    </div>
  );
};

export default JobsPage;
