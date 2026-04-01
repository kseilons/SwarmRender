import React, { useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { jobService } from '../services/api';
import { wsService } from '../services/ws';
import { ChevronLeft, Cpu, HardDrive, Layout, Maximize2, RefreshCw, RotateCcw } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

import { useTranslation } from '../context/LanguageContext';

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const { t } = useTranslation();
  const statusClass = status.toLowerCase() || 'created';

  return (
    <span className={`status-badge status-${statusClass} ${status === 'PROCESSING' ? 'pulsing' : ''}`}>
      {t(`status.${status}`)}
    </span>
  );
};

const JobDetailsPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  
  const { data, isLoading, error } = useQuery({
    queryKey: ['job', id],
    queryFn: () => jobService.getJob(id!),
    enabled: !!id,
  });

  const { job, tasks } = data || { job: null, tasks: [] };

  const retryMutation = useMutation({
    mutationFn: () => jobService.retryJob(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['job', id] });
    },
  });

  const hasFailedTasks = tasks?.some(t => t.status === 'FAILED');
  const canRetry = job?.status === 'FAILED' || hasFailedTasks;

  // WebSocket Subscription
  useEffect(() => {
    const unsubscribe = wsService.subscribe((msg) => {
      if (msg.type === 'task_update' && msg.task_id) {
        // Optimistically update the UI or refetch
        queryClient.invalidateQueries({ queryKey: ['job', id] });
      }
      if (msg.type === 'job_update' && msg.job_id === id) {
        queryClient.invalidateQueries({ queryKey: ['job', id] });
      }
    });
    return () => unsubscribe();
  }, [id, queryClient]);

  if (isLoading) return <div className="p-20 text-center animate-pulse text-white/20 uppercase tracking-widest font-bold">{t('common.loading')}</div>;
  if (error || !data || !job) return <div className="p-20 text-center text-rose-400">{t('common.error_loading')}</div>;

  return (
    <motion.div 
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="space-y-8"
    >
      {/* Breadcrumbs & Actions */}
      <div className="flex items-center justify-between">
        <Link to="/" className="flex items-center gap-2 text-white/40 hover:text-white transition-colors text-sm font-medium">
          <ChevronLeft size={18} /> {t('common.back')}
        </Link>
        <div className="flex gap-2">
          {canRetry && (
            <button 
              onClick={() => retryMutation.mutate()}
              disabled={retryMutation.isPending}
              className="glass glass-interactive px-4 py-2 text-rose-400 hover:text-rose-300 border-none bg-rose-500/10 cursor-pointer flex items-center gap-2 text-sm font-bold"
            >
              {retryMutation.isPending ? <RefreshCw size={18} className="animate-spin" /> : <RotateCcw size={18} />}
              {t('common.retry')}
            </button>
          )}
          <button 
            onClick={() => queryClient.invalidateQueries({ queryKey: ['job', id] })}
            className="glass glass-interactive p-2 text-white/60 hover:text-white border-none bg-transparent cursor-pointer"
          >
            <RefreshCw size={18} />
          </button>
        </div>
      </div>

      {/* Hero Section */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-6">
          <div className="glass p-8 flex flex-col md:flex-row gap-8">
            <div className="w-full md:w-64 h-48 bg-zinc-900 rounded-xl overflow-hidden glass border-white/5 relative">
              {job.preview_url ? (
                  <img src={job.preview_url} className="w-full h-full object-cover" />
              ) : (
                  <div className="w-full h-full flex items-center justify-center text-white/10 italic text-xs">{t('common.rendering_preview')}</div>
              )}
              <div className="absolute top-2 right-2">
                 <button className="bg-black/50 p-1.5 rounded-lg hover:bg-black transition-colors border-none text-white cursor-pointer"><Maximize2 size={14} /></button>
              </div>
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <h1 className="text-4xl font-bold tracking-tight">{job.name}</h1>
                <StatusBadge status={job.status} />
              </div>
              <p className="text-white/40 font-mono text-xs mb-6 uppercase tracking-widest">{job.id}</p>
              
              <div className="grid grid-cols-2 md:grid-cols-3 gap-6">
                <div className="space-y-1">
                  <span className="text-[10px] text-zinc-500 uppercase font-bold tracking-tighter">{t('job_details.engine')}</span>
                  <p className="flex items-center gap-2 font-semibold text-zinc-300"><Cpu size={16} className="text-cyan-500" /> {job.options.engine || 'EEVEE'}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-[10px] text-zinc-500 uppercase font-bold tracking-tighter">{t('job_details.frame_range')}</span>
                  <p className="flex items-center gap-2 font-semibold text-zinc-300"><Layout size={16} className="text-purple-500" /> {job.frame_start} — {job.frame_end}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-[10px] text-zinc-500 uppercase font-bold tracking-tighter">{t('job_details.allocation')}</span>
                  <p className="flex items-center gap-2 font-semibold text-zinc-300"><HardDrive size={16} className="text-emerald-500" /> {job.worker_count} {t('common.workers')}</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Quick Stats Sidebar */}
        <div className="space-y-4">
           <div className="glass p-6 bg-cyan-500/5 border-cyan-500/20">
              <h3 className="text-sm font-bold uppercase tracking-widest text-cyan-400 mb-4 opacity-70">{t('job_details.progress_trace')}</h3>
              <div className="flex items-end gap-1 h-20">
                {tasks.map((task: any, i: number) => (
                    <div key={i} className={`flex-1 rounded-sm w-full h-${Math.random() > 0.5 ? 'full' : '1/2'} ${task.status === 'COMPLETED' ? 'bg-emerald-500' : task.status === 'PROCESSING' ? 'bg-cyan-500 pulsing' : 'bg-white/5'}`}></div>
                ))}
              </div>
              <div className="flex justify-between mt-4 text-[10px] uppercase font-bold text-white/30 tracking-tighter">
                <span>0%</span>
                <span>{t('job_details.active_graph')}</span>
                <span>100%</span>
              </div>
           </div>
        </div>
      </div>

      {/* Tasks Table */}
      <div className="space-y-4">
        <h2 className="text-2xl font-bold tracking-tight">{t('job_details.cluster_title')}</h2>
        <div className="glass overflow-hidden">
          <table className="w-full text-left">
            <thead>
              <tr className="bg-white/5 text-[10px] uppercase tracking-widest text-white/40">
                <th className="px-6 py-4 font-bold">{t('job_details.th_node')}</th>
                <th className="px-6 py-4 font-bold">{t('job_details.th_frames')}</th>
                <th className="px-6 py-4 font-bold">{t('job_details.th_state')}</th>
                <th className="px-6 py-4 font-bold">{t('job_details.th_worker')}</th>
                <th className="px-6 py-4 font-bold text-right">{t('job_details.th_sync')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              <AnimatePresence>
                {tasks.map((task) => (
                  <motion.tr 
                    key={task.id}
                    layout
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="hover:bg-white/5 transition-colors group"
                  >
                    <td className="px-6 py-4">
                      <div className="flex flex-col">
                        <span className="text-sm font-mono text-white/90 truncate w-32">{task.id}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 underline decoration-white/10 decoration-wavy underline-offset-4 font-medium text-zinc-300">
                      {task.frame_start} — {task.frame_end}
                    </td>
                    <td className="px-6 py-4">
                      <StatusBadge status={task.status} />
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <div className={`w-2 h-2 rounded-full ${task.worker_id ? 'bg-emerald-500' : 'bg-white/10'}`}></div>
                        <div className="flex flex-col">
                           <span className="text-xs font-mono text-white/40">{task.worker_id || t('common.unassigned')}</span>
                           {task.retry_count > 0 && (
                             <span className="text-[9px] text-rose-500/50 uppercase font-bold">Retries: {task.retry_count}</span>
                           )}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-right text-xs text-white/20 font-mono">
                      {new Date(task.updated_at).toLocaleTimeString()}
                    </td>
                  </motion.tr>
                ))}
              </AnimatePresence>
            </tbody>
          </table>
        </div>
      </div>
    </motion.div>
  );
};

export default JobDetailsPage;
