import React, { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { jobService, Job } from '../services/api';
import { useNavigate } from 'react-router-dom';
import { Clock, HardDrive, Layers, ChevronRight, Play } from 'lucide-react';
import { motion } from 'framer-motion';
import { useTranslation } from '../context/LanguageContext';
import { wsService } from '../services/ws';

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
          <div className="preview-placeholder">
            <Layers size={48} />
            <span className="preview-placeholder-text">{t('common.rendering_preview')}</span>
          </div>
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-60"></div>
        <div className="absolute top-3 left-3">
          <StatusBadge status={job.status} />
        </div>
        <div className="absolute bottom-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity">
          <div className="w-8 h-8 bg-cyan-500 rounded-full flex items-center justify-center text-black">
            <Play size={14} fill="currentColor" />
          </div>
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

        <div className="mt-4 pt-4 border-t border-white/5 flex items-center justify-between">
          <div className="flex items-center gap-2 text-white/40 text-[10px]">
             <Clock size={12} /> {new Date(job.created_at).toLocaleDateString()}
          </div>
          <ChevronRight size={16} className="text-white/20 group-hover:text-white group-hover:translate-x-1 transition-all" />
        </div>
      </div>
    </motion.div>
  );
};

const JobsPage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data: jobs, isLoading, error } = useQuery({
    queryKey: ['jobs'],
    queryFn: jobService.listJobs,
  });

  // WebSocket for real-time updates
  useEffect(() => {
    const unsubscribe = wsService.subscribe((msg) => {
      if (msg.type === 'job_update') {
        queryClient.invalidateQueries({ queryKey: ['jobs'] });
      }
    });
    return () => unsubscribe();
  }, [queryClient]);

  if (isLoading) return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
      {[1,2,3,4].map(i => (
        <div key={i} className="glass h-64 animate-pulse bg-white/5 border-white/5"></div>
      ))}
    </div>
  );

  if (error) return (
    <div className="glass p-8 text-center bg-rose-500/5 border-rose-500/20">
      <h2 className="text-xl font-bold text-rose-400">{t('common.error_loading')}</h2>
    </div>
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
        {jobs?.map((job) => (
          <JobCard key={job.id} job={job} />
        ))}
      </div>
      
      {jobs?.length === 0 && (
        <div className="glass p-16 text-center border-dashed border-2 bg-transparent opacity-50">
          <div className="w-16 h-16 bg-white/5 rounded-full flex items-center justify-center mx-auto mb-4">
            <Layers size={32} className="text-white/20" />
          </div>
          <h3 className="text-xl font-semibold">{t('common.no_active_jobs')}</h3>
          <p className="mt-2 text-sm text-white/40">{t('common.add_first_project')}</p>
        </div>
      )}
    </div>
  );
};

export default JobsPage;
