import React, { useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { jobService } from '../services/api';
import { wsService } from '../services/ws';
import { ChevronLeft, Cpu, HardDrive, Layout, Maximize2, RefreshCw, RotateCcw, Layers, Download } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import ProgressBar from '../components/common/ProgressBar';
import PreviewPlaceholder from '../components/common/PreviewPlaceholder';

import { useTranslation } from '../context/LanguageContext';

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const { t } = useTranslation();
  const statusClass = status.toLowerCase() || 'created';

  return (
    <span className={`status-badge status-${statusClass} ${status === 'PROCESSING' ? 'pulsing' : ''} shadow-lg shadow-current/5`}>
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
      if (msg.type === 'task_update' || msg.type === 'task_progress') {
        queryClient.invalidateQueries({ queryKey: ['job', id] });
      }
      if ((msg.type === 'job_update' || msg.type === 'job_progress') && msg.job_id === id) {
        queryClient.invalidateQueries({ queryKey: ['job', id] });
      }
    });
    return () => unsubscribe();
  }, [id, queryClient]);

  if (isLoading) return (
    <div className="p-20 flex flex-col items-center justify-center space-y-4">
      <RefreshCw size={48} className="animate-spin text-white/10" />
      <div className="animate-pulse text-white/20 uppercase tracking-widest font-bold text-sm">
        {t('common.loading')}
      </div>
    </div>
  );

  if (error || !data || !job) return (
    <div className="p-20 flex flex-col items-center justify-center space-y-6">
      <div className="w-16 h-16 rounded-full bg-rose-500/10 flex items-center justify-center text-rose-500">
        <RotateCcw size={32} />
      </div>
      <div className="text-center space-y-2">
        <h2 className="text-2xl font-bold text-white">{t('common.error_loading')}</h2>
        <p className="text-white/40 text-sm max-w-xs">{t('common.error_loading_desc') || 'We couldn\'t retrieve the job details. Please check your connection or try again.'}</p>
      </div>
      <button
        onClick={() => queryClient.invalidateQueries({ queryKey: ['job', id] })}
        className="glass glass-interactive px-6 py-3 text-white font-bold rounded-xl border-none bg-white/5"
      >
        {t('common.retry')}
      </button>
    </div>
  );

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="space-y-8 p-4 md:p-8"
    >
      {/* Header Actions */}
      <div className="flex items-center justify-between">
        <Link to="/" className="group flex items-center gap-2 text-white/40 hover:text-white transition-colors text-sm font-medium">
          <div className="w-8 h-8 rounded-lg bg-white/5 flex items-center justify-center group-hover:bg-white/10 transition-colors">
            <ChevronLeft size={18} />
          </div>
          {t('common.back')}
        </Link>

        <div className="flex gap-3">
          {job?.status === 'COMPLETED' && job?.result_url && (
            <a
              href={job.result_url}
              target="_blank"
              rel="noopener noreferrer"
              className="glass glass-interactive px-6 py-2.5 text-black hover:text-zinc-900 border-none bg-emerald-400 cursor-pointer flex items-center gap-2 text-sm font-bold rounded-xl shadow-[0_0_20px_rgba(16,185,129,0.3)] transition-all hover:scale-105 no-underline"
            >
              <Download size={18} />
              {t('common.download_result')}
            </a>
          )}
          {canRetry && (
            <button
              onClick={() => retryMutation.mutate()}
              disabled={retryMutation.isPending}
              className="glass glass-interactive px-6 py-2.5 text-black hover:text-black border-none bg-cyan-400 cursor-pointer flex items-center gap-2 text-sm font-bold rounded-xl shadow-[0_0_20px_rgba(34,211,238,0.3)] transition-all hover:scale-105"
            >
              {retryMutation.isPending ? <RefreshCw size={18} className="animate-spin" /> : <RotateCcw size={18} />}
              {t('common.retry_job')}
            </button>
          )}
          <button
            onClick={() => queryClient.invalidateQueries({ queryKey: ['job', id] })}
            className="glass glass-interactive p-2.5 text-white/60 hover:text-white border-none bg-white/5 cursor-pointer rounded-xl"
            title={t('common.refresh')}
          >
            <RefreshCw size={20} />
          </button>
        </div>
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Side: Job Info */}
        <div className="lg:col-span-2">
          <div className="glass p-8 relative overflow-hidden h-full">
            {/* Background Decoration */}
            <div className="absolute top-0 right-0 w-64 h-64 bg-cyan-500/5 rounded-full blur-3xl -mr-32 -mt-32"></div>

            <div className="flex flex-col md:flex-row gap-8 relative z-10">
              {/* Preview Image / Video */}
              <div className="w-full md:w-72 aspect-video md:aspect-square bg-zinc-900 rounded-2xl overflow-hidden glass border-white/5 group relative flex-shrink-0">
                {job.status === 'COMPLETED' && job.result_url?.toLowerCase().split('?')[0].endsWith('.mp4') ? (
                  <video
                    src={job.result_url}
                    controls
                    className="w-full h-full object-cover"
                    poster={job.preview_url}
                  />
                ) : job.preview_url ? (
                  <img src={job.preview_url} className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-110" alt="Job Preview" />
                ) : (
                  <PreviewPlaceholder name={job.name} size="lg" />
                )}
                <div className="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button className="bg-black/60 backdrop-blur-md p-2 rounded-xl hover:bg-black transition-colors border-none text-white cursor-pointer shadow-xl">
                    <Maximize2 size={16} />
                  </button>
                </div>
                {job.status === 'COMPLETED' && job.result_url && (
                  <div className="absolute inset-x-0 bottom-0 p-4 bg-gradient-to-t from-black/80 to-transparent">
                    <a
                      href={job.result_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="w-full py-2 bg-emerald-500 hover:bg-emerald-400 text-black text-[10px] font-black uppercase tracking-widest rounded-lg flex items-center justify-center gap-2 transition-colors no-underline"
                    >
                      <Download size={14} /> {t('common.download_result')}
                    </a>
                  </div>
                )}
              </div>

              {/* Info Details */}
              <div className="flex-1 flex flex-col min-w-0">
                <div className="flex flex-wrap items-center gap-3 mb-6">
                  <h1 className="text-3xl md:text-5xl font-bold tracking-tight text-white m-0 truncate">{job.name}</h1>
                  <StatusBadge status={job.status} />
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-16 gap-y-10 mt-6">
                  <div className="space-y-4">
                    <span className="text-[10px] text-zinc-500 uppercase font-black tracking-widest block opacity-60">{t('job_details.engine')}</span>
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-2xl bg-cyan-500/10 flex items-center justify-center text-cyan-400 border border-cyan-500/10">
                        <Cpu size={24} />
                      </div>
                      <span className="font-bold text-zinc-100 text-2xl tracking-tight">{job.options.engine || 'EEVEE'}</span>
                    </div>
                  </div>

                  <div className="space-y-4">
                    <span className="text-[10px] text-zinc-500 uppercase font-black tracking-widest block opacity-60">{t('job_details.frame_range')}</span>
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-2xl bg-purple-500/10 flex items-center justify-center text-purple-400 border border-purple-500/10">
                        <Layout size={24} />
                      </div>
                      <span className="font-bold text-zinc-100 text-2xl tracking-tight">{job.frame_start} — {job.frame_end}</span>
                    </div>
                  </div>

                  <div className="space-y-4">
                    <span className="text-[10px] text-zinc-500 uppercase font-black tracking-widest block opacity-60">{t('job_details.allocation')}</span>
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 flex items-center justify-center text-emerald-400 border border-emerald-500/10">
                        <HardDrive size={24} />
                      </div>
                      <div className="flex flex-col">
                        <span className="font-bold text-zinc-100 text-2xl tracking-tight leading-none">{job.worker_count}</span>
                        <span className="text-[10px] uppercase tracking-wider text-zinc-500 font-bold mt-1.5">{t('common.workers')}</span>
                      </div>
                    </div>
                  </div>

                  {job.options.scene && (
                    <div className="space-y-4">
                      <span className="text-[10px] text-zinc-500 uppercase font-black tracking-widest block opacity-60">{t('job_details.scene_name')}</span>
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 rounded-2xl bg-amber-500/10 flex items-center justify-center text-amber-500 border border-amber-500/10">
                          <Layers size={24} />
                        </div>
                        <span className="font-bold text-zinc-100 text-xl tracking-tight truncate max-w-[200px]">{job.options.scene}</span>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Right Side: Visual Progress */}
        <div className="h-full">
          <div className="glass p-8 h-full flex flex-col bg-[#0a0a0f] border-cyan-500/10 relative overflow-hidden">
            <div className="absolute top-0 right-0 w-32 h-32 bg-cyan-500/5 blur-2xl rounded-full -mr-16 -mt-16"></div>

            <div className="flex items-center justify-between mb-10 relative z-10">
              <h3 className="text-[11px] font-black uppercase tracking-[0.2em] text-cyan-400">{t('job_details.progress_trace')}</h3>
              <RefreshCw size={14} className={`text-cyan-400/40 ${job.status === 'PROCESSING' ? 'animate-spin' : ''}`} />
            </div>

            <div className="flex items-end gap-2 h-48 mb-8 relative z-10 px-2">
              {tasks.map((task: any, i: number) => {
                const isCompleted = task.status === 'COMPLETED';
                const isProcessing = task.status === 'PROCESSING';
                const isFailed = task.status === 'FAILED';

                const taskTotal = task.frame_end - task.frame_start + 1;
                const taskCurrent = task.status === 'COMPLETED' ? taskTotal :
                  (task.current_frame && task.current_frame >= task.frame_start ? (task.current_frame - task.frame_start + 1) : 0);
                const progressRatio = taskTotal > 0 ? taskCurrent / taskTotal : 0;

                return (
                  <motion.div
                    key={i}
                    initial={{ height: 4 }}
                    animate={{ height: isCompleted ? '100%' : isProcessing ? `${Math.max(20, progressRatio * 100)}%` : '15%' }}
                    className={`flex-1 rounded-t-lg transition-all duration-700 min-h-[6px] ${isCompleted ? 'bg-emerald-500 shadow-[0_-5px_15px_rgba(16,185,129,0.2)]' :
                      isProcessing ? 'bg-cyan-500 pulsing shadow-[0_-5px_15px_rgba(6,182,212,0.2)]' :
                        isFailed ? 'bg-rose-500 opacity-50' :
                          'bg-white/10'
                      }`}
                  >
                    {isProcessing && progressRatio > 0.1 && (
                      <div className="absolute top-2 left-1/2 -translate-x-1/2 text-[8px] font-black text-black/60 bg-white/40 px-1 rounded-sm">
                        {Math.round(progressRatio * 100)}%
                      </div>
                    )}
                  </motion.div>
                );
              })}
            </div>

            <div className="mt-auto relative z-10">
              <div className="flex justify-between items-center text-[10px] uppercase font-black text-white/20 tracking-widest pt-6 border-t border-white/5">
                <div className="flex flex-col gap-1">
                  <span className="text-white/60 font-mono">0%</span>
                  <span>{t('common.start')}</span>
                </div>
                <div className="flex flex-col items-end gap-1">
                  <span className="text-white/60 font-mono">100%</span>
                  <span>{t('common.end')}</span>
                </div>
              </div>
              <div className="text-center mt-8">
                <span className="text-[10px] text-cyan-400/30 font-black uppercase tracking-[0.2em]">{t('job_details.realtime_cluster_metrics')}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Cluster Nodes Section */}
      <div className="space-y-6 pt-6">
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 px-2">
          <div className="space-y-2">
            <h2 className="text-2xl md:text-3xl font-bold tracking-tight text-white">{t('job_details.cluster_title')}</h2>
            <p className="text-white/40 text-sm max-w-xl leading-relaxed">{t('job_details.cluster_desc')}</p>
          </div>
          <div className="flex items-center gap-8 text-[8px] font-black tracking-[0.25em] uppercase text-white/20 bg-white/[0.02] px-6 py-3 rounded-full border border-white/5">
            <span className="flex items-center gap-3.5"><div className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.3)]"></div> {t('status.COMPLETED')}</span>
            <span className="flex items-center gap-3.5"><div className="w-2 h-2 rounded-full bg-cyan-500 shadow-[0_0_8px_rgba(6,182,212,0.3)]"></div> {t('status.PROCESSING')}</span>
            <span className="flex items-center gap-3.5"><div className="w-2 h-2 rounded-full bg-white/10"></div> {t('status.CREATED')}</span>
          </div>
        </div>

        <div className="glass overflow-hidden border-white/5 shadow-2xl bg-black/40">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-white/[0.02] text-[10px] uppercase tracking-[0.2em] text-white/25">
                  <th className="px-8 py-5 font-black border-b border-white/5">{t('job_details.th_node')}</th>
                  <th className="px-8 py-5 font-black border-b border-white/5 text-center">{t('job_details.th_type')}</th>
                  <th className="px-8 py-5 font-black border-b border-white/5 text-center">{t('job_details.th_frames')}</th>
                  <th className="px-8 py-5 font-black border-b border-white/5">{t('job_details.th_progress') || 'Progress'}</th>
                  <th className="px-8 py-5 font-black border-b border-white/5 text-center">{t('job_details.th_state')}</th>
                  <th className="px-8 py-5 font-black border-b border-white/5">{t('job_details.th_worker')}</th>
                  <th className="px-8 py-5 font-black text-right border-b border-white/5">{t('job_details.th_sync')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                <AnimatePresence mode="popLayout">
                  {tasks.map((task) => (
                    <motion.tr
                      key={task.id}
                      layout
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      className="hover:bg-white/[0.04] transition-colors group"
                    >
                      <td className="px-8 py-6">
                        <div className="flex items-center gap-6">
                          <div className={`w-1 h-10 rounded-full transition-shadow duration-500 ${task.status === 'COMPLETED' ? 'bg-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.4)]' : task.status === 'PROCESSING' ? 'bg-cyan-500 pulsing shadow-[0_0_12px_rgba(6,182,212,0.4)]' : 'bg-white/10'}`}></div>
                          <div className="flex flex-col gap-1.5">
                            <span className="text-[11px] font-mono text-white/70 tracking-tight group-hover:text-cyan-400 transition-colors uppercase">{task.id.split('-')[0]}</span>
                          </div>
                        </div>
                      </td>
                      <td className="px-8 py-6 text-center">
                        <span className="text-[10px] font-semibold tracking-[0.2em] text-white/40 uppercase">
                          {task.type}
                        </span>
                      </td>
                      <td className="px-8 py-6 text-center">
                        <div className="inline-flex items-center px-4 py-1.5 rounded-xl bg-white/[0.03] border border-white/5 font-mono text-zinc-400 text-xs shadow-inner">
                          {task.frame_start} <span className="mx-2 text-white/10">—</span> {task.frame_end}
                        </div>
                      </td>
                      <td className="px-8 py-6 min-w-[150px]">
                        <ProgressBar
                          current={task.status === 'COMPLETED' ? (task.frame_end - task.frame_start + 1) : (task.current_frame && task.current_frame >= task.frame_start ? (task.current_frame - task.frame_start + 1) : 0)}
                          total={task.frame_end - task.frame_start + 1}
                          height={4}
                          showStats={true}
                        />
                      </td>
                      <td className="px-8 py-6 text-center">
                        <StatusBadge status={task.status} />
                      </td>
                      <td className="px-8 py-6">
                        <div className="flex items-center gap-4">
                          <div className={`w-10 h-10 rounded-xl flex items-center justify-center transition-all border ${task.worker_id ? 'bg-emerald-500/10 text-emerald-500 border-emerald-500/10 shadow-[0_0_15px_rgba(16,185,129,0.1)]' : 'bg-white/5 text-white/10 border-white/5'}`}>
                            {task.worker_id ? <Cpu size={20} /> : <HardDrive size={20} />}
                          </div>
                          <div className="flex flex-col">
                            <span className="text-xs font-mono font-medium text-white/60">{task.worker_id || t('common.unassigned')}</span>
                            {task.retry_count > 0 && (
                              <span className="text-[9px] text-rose-500/80 uppercase font-black tracking-[0.2em] mt-1">
                                {t('common.retries')}: {task.retry_count}
                              </span>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-8 py-6 text-right">
                        <div className="flex flex-col items-end gap-1">
                          <span className="text-[10px] text-white/80 font-mono font-medium tracking-tight">{new Date(task.updated_at || task.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
                          <span className="text-[10px] text-white/20 uppercase font-bold tracking-tight">{new Date(task.updated_at || task.created_at).toLocaleDateString()}</span>
                        </div>
                      </td>
                    </motion.tr>
                  ))}
                </AnimatePresence>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </motion.div>
  );
};

export default JobDetailsPage;
