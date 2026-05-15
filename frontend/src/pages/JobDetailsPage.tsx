import React, { useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { jobService } from '../services/api';
import { wsService } from '../services/ws';
import { ChevronLeft, Cpu, HardDrive, Layout, RefreshCw, RotateCcw, Layers, Download, FileText, X, Clock, Calendar, CheckCircle } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import ProgressBar from '../components/common/ProgressBar';
import PreviewPlaceholder from '../components/common/PreviewPlaceholder';

import { useTranslation } from '../context/LanguageContext';

const LogModal: React.FC<{ logs: string; onClose: () => void; taskID: string }> = ({ logs, onClose, taskID }) => {
  const { t } = useTranslation();

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 z-[100] flex items-center justify-center p-4 md:p-8 bg-black/80 backdrop-blur-sm"
      onClick={onClose}
    >
      <motion.div
        initial={{ scale: 0.9, y: 20 }}
        animate={{ scale: 1, y: 0 }}
        className="glass w-full max-w-4xl max-h-[80vh] flex flex-col overflow-hidden bg-zinc-950 border-white/5 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6 border-b border-white/5 flex items-center justify-between bg-white/[0.02]">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-rose-500/10 flex items-center justify-center text-rose-500">
              <FileText size={20} />
            </div>
            <div>
              <h3 className="text-lg font-bold text-white leading-tight">{t('job_details.blender_logs')}</h3>
              <p className="text-xs text-white/40 font-mono">Task ID: {taskID}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-white/5 rounded-lg text-white/40 hover:text-white transition-colors"
          >
            <X size={20} />
          </button>
        </div>
        <div className="flex-1 overflow-auto p-6 font-mono text-[13px] leading-relaxed text-zinc-300 selection:bg-cyan-500/30">
          <pre className="whitespace-pre-wrap break-all">
            {logs || 'No logs available for this task.'}
          </pre>
        </div>
        <div className="p-4 border-t border-white/5 bg-white/[0.01] flex justify-end">
          <button
            onClick={onClose}
            className="px-6 py-2 bg-white/5 hover:bg-white/10 text-white text-sm font-bold rounded-xl transition-colors border border-white/5"
          >
            {t('common.close') || 'Close'}
          </button>
        </div>
      </motion.div>
    </motion.div>
  );
};

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
  const [selectedTaskLogs, setSelectedTaskLogs] = React.useState<{ logs: string; id: string } | null>(null);

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

  const handleDownload = async (url: string, filename: string) => {
    try {
      const response = await fetch(url);
      const blob = await response.blob();
      const blobUrl = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = blobUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(blobUrl);
    } catch (error) {
      console.error('Download failed:', error);
      window.open(url, '_blank');
    }
  };

  const formatDuration = (start: string, end?: string) => {
    if (!end) return null;
    const diff = new Date(end).getTime() - new Date(start).getTime();
    if (diff <= 0) return '0s';
    
    const h = Math.floor(diff / 3600000);
    const m = Math.floor((diff % 3600000) / 60000);
    const s = Math.floor((diff % 60000) / 1000);
    
    return [
      h > 0 ? `${h}h` : '',
      m > 0 ? `${m}m` : '',
      `${s}s`
    ].filter(Boolean).join(' ');
  };

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
            <button
              onClick={() => handleDownload(job.result_url!, `${job.name.replace(/\s+/g, '_')}_result.mp4`)}
              className="glass glass-interactive px-6 py-2.5 text-black hover:text-zinc-900 border-none bg-emerald-400 cursor-pointer flex items-center gap-2 text-sm font-bold rounded-xl shadow-[0_0_20px_rgba(16,185,129,0.3)] transition-all hover:scale-105"
            >
              <Download size={18} />
              {t('common.download_result')}
            </button>
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

      {/* Project Header - Full Width */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 mb-2">
        <div className="flex flex-wrap items-center gap-4">
          <h1 className="text-3xl md:text-5xl font-bold tracking-tight text-white m-0">{job.name}</h1>
          <StatusBadge status={job.status} />
        </div>
      </div>

      {/* Main Content Grid: Balanced 2/1 Ratio */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Side: Large Preview Area (2/3 width) */}
        <div className="lg:col-span-2">
          <div className="glass p-6 relative overflow-hidden h-full flex flex-col items-center justify-center min-h-[400px]">
            {/* Background Decoration */}
            <div className="absolute top-0 right-0 w-64 h-64 bg-cyan-500/5 rounded-full blur-3xl -mr-32 -mt-32"></div>

            {job.status === 'COMPLETED' && job.result_url?.toLowerCase().split('?')[0].endsWith('.mp4') ? (
              <div className="w-full aspect-video bg-zinc-900/50 rounded-2xl overflow-hidden glass border-white/5 group relative flex-shrink-0 flex items-center justify-center">
                <video
                  src={job.result_url}
                  controls
                  className="w-full h-full object-cover"
                  poster={job.preview_url}
                />
              </div>
            ) : job.preview_url ? (
              <div className="w-full aspect-video bg-zinc-900/50 rounded-2xl overflow-hidden glass border-white/5 group relative flex-shrink-0 flex items-center justify-center">
                <img src={job.preview_url} className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-110" alt="Job Preview" />
              </div>
            ) : (
              <div className="w-full h-full aspect-video bg-zinc-900/50 rounded-2xl overflow-hidden glass border-white/5 group relative flex-shrink-0 flex items-center justify-center">
                <PreviewPlaceholder name={job.name} size="lg" />
              </div>
            )}
          </div>
        </div>

        {/* Right Side: Meta Info & Progress (1/3 width, split vertically) */}
        <div className="lg:col-span-1 space-y-8">
          {/* Metadata Card */}
          <div className="glass p-6 relative overflow-hidden bg-white/[0.01] border-white/5 shadow-2xl">
            <div className="absolute top-0 left-0 w-32 h-32 bg-cyan-500/5 blur-2xl rounded-full -ml-16 -mt-16"></div>

            <div className="space-y-6 relative z-10">
              {/* Scene Name at the Top */}
              {job.options.scene && (
                <div className="flex items-center gap-4 group p-3 -m-3 rounded-2xl hover:bg-white/[0.03] transition-colors">
                  <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-white/40 group-hover:text-amber-500 group-hover:bg-amber-500/10 transition-all border border-white/5 shadow-inner">
                    <Layers size={18} />
                  </div>
                  <div className="flex flex-col">
                    <span className="text-[10px] text-white/30 uppercase font-black tracking-[0.15em] leading-none mb-1.5">{t('job_details.scene_name')}</span>
                    <span className="font-bold text-white text-lg tracking-tight truncate leading-tight group-hover:text-amber-500 transition-colors">{job.options.scene}</span>
                  </div>
                </div>
              )}

              <div className="grid grid-cols-1 gap-6 pt-2 border-t border-white/5">
                <div className="flex items-center gap-4 group p-3 -m-3 rounded-2xl hover:bg-white/[0.03] transition-colors">
                  <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-white/40 group-hover:text-cyan-400 group-hover:bg-cyan-500/10 transition-all border border-white/5 shadow-inner">
                    <Cpu size={18} />
                  </div>
                  <div className="flex flex-col">
                    <span className="text-[10px] text-white/30 uppercase font-black tracking-[0.15em] leading-none mb-1.5">{t('job_details.engine')}</span>
                    <span className="font-bold text-white text-lg tracking-tight uppercase leading-tight group-hover:text-cyan-400 transition-colors">{job.options.engine || 'EEVEE'}</span>
                  </div>
                </div>

                <div className="flex items-center gap-4 group p-3 -m-3 rounded-2xl hover:bg-white/[0.03] transition-colors">
                  <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-white/40 group-hover:text-purple-400 group-hover:bg-purple-500/10 transition-all border border-white/5 shadow-inner">
                    <Layout size={18} />
                  </div>
                  <div className="flex flex-col">
                    <span className="text-[10px] text-white/30 uppercase font-black tracking-[0.15em] leading-none mb-1.5">{t('job_details.frame_range')}</span>
                    <span className="font-bold text-white text-lg tracking-tight leading-tight group-hover:text-purple-400 transition-colors">{job.frame_start} — {job.frame_end}</span>
                  </div>
                </div>

                <div className="flex items-center gap-4 group p-3 -m-3 rounded-2xl hover:bg-white/[0.03] transition-colors">
                  <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-white/40 group-hover:text-emerald-400 group-hover:bg-emerald-500/10 transition-all border border-white/5 shadow-inner">
                    <HardDrive size={18} />
                  </div>
                  <div className="flex flex-col">
                    <span className="text-[10px] text-white/30 uppercase font-black tracking-[0.15em] leading-none mb-1.5">{t('job_details.allocation')}</span>
                    <div className="flex items-baseline gap-2 leading-tight">
                      <span className="font-bold text-white text-lg tracking-tight mr-1 group-hover:text-emerald-400 transition-colors">{job.worker_count}</span>
                      <span className="text-[9px] uppercase tracking-widest text-white/20 font-black">{t('common.workers')}</span>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-4 group p-3 -m-3 rounded-2xl hover:bg-white/[0.03] transition-colors">
                  <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-white/40 group-hover:text-amber-400 group-hover:bg-amber-500/10 transition-all border border-white/5 shadow-inner">
                    <Calendar size={18} />
                  </div>
                  <div className="flex flex-col">
                    <span className="text-[10px] text-white/30 uppercase font-black tracking-[0.15em] leading-none mb-1.5">{t('job_details.created')}</span>
                    <span className="font-bold text-white text-lg tracking-tight leading-tight group-hover:text-amber-400 transition-colors">
                      {new Date(job.created_at).toLocaleString([], { dateStyle: 'short', timeStyle: 'short' })}
                    </span>
                  </div>
                </div>

                {job.completed_at && (
                  <>
                    <div className="flex items-center gap-4 group p-3 -m-3 rounded-2xl hover:bg-white/[0.03] transition-colors">
                      <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-white/40 group-hover:text-emerald-400 group-hover:bg-emerald-500/10 transition-all border border-white/5 shadow-inner">
                        <CheckCircle size={18} />
                      </div>
                      <div className="flex flex-col">
                        <span className="text-[10px] text-white/30 uppercase font-black tracking-[0.15em] leading-none mb-1.5">{t('job_details.completed')}</span>
                        <span className="font-bold text-white text-lg tracking-tight leading-tight group-hover:text-emerald-400 transition-colors">
                          {new Date(job.completed_at).toLocaleString([], { dateStyle: 'short', timeStyle: 'short' })}
                        </span>
                      </div>
                    </div>

                    <div className="flex items-center gap-4 group p-3 -m-3 rounded-2xl hover:bg-white/[0.03] transition-colors">
                      <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-white/40 group-hover:text-cyan-400 group-hover:bg-cyan-500/10 transition-all border border-white/5 shadow-inner">
                        <Clock size={18} />
                      </div>
                      <div className="flex flex-col">
                        <span className="text-[10px] text-white/30 uppercase font-black tracking-[0.15em] leading-none mb-1.5">{t('job_details.duration')}</span>
                        <span className="font-bold text-white text-lg tracking-tight leading-tight group-hover:text-cyan-400 transition-colors">
                          {formatDuration(job.created_at, job.completed_at)}
                        </span>
                      </div>
                    </div>
                  </>
                )}
              </div>
            </div>
          </div>

          {/* Progress Card */}
          <div className="glass p-8 flex flex-col bg-[#0a0a0f] border-cyan-500/10 relative overflow-hidden min-h-[350px]">
            <div className="absolute top-0 right-0 w-32 h-32 bg-cyan-500/5 blur-2xl rounded-full -mr-16 -mt-16"></div>

            <div className="flex items-center justify-between mb-8 relative z-10">
              <h3 className="text-[11px] font-black uppercase tracking-[0.2em] text-cyan-400">{t('job_details.progress_trace')}</h3>
              <RefreshCw size={14} className={`text-cyan-400/40 ${job.status === 'PROCESSING' ? 'animate-spin' : ''}`} />
            </div>

            <div className="flex items-end gap-1.5 h-32 mb-8 relative z-10 px-1">
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
                    className={`flex-1 rounded-t-sm transition-all duration-700 min-h-[4px] ${isCompleted ? 'bg-emerald-500 shadow-[0_-5px_15px_rgba(16,185,129,0.2)]' :
                      isProcessing ? 'bg-cyan-500 pulsing shadow-[0_-5px_15px_rgba(6,182,212,0.2)]' :
                        isFailed ? 'bg-rose-500 opacity-50' :
                          'bg-white/5'
                      }`}
                  />
                );
              })}
            </div>

            <div className="mt-auto relative z-10">
              <div className="mt-8 space-y-4">
                <div className="flex justify-between items-center px-1">
                  <span className="text-[10px] text-white/40 uppercase font-black tracking-widest">{t('job_details.completed_frames')}</span>
                  <span className="text-xl font-bold text-white tabular-nums">
                    {job.completed_frames} <span className="text-white/20 text-xs ml-1 font-medium">/ {job.frame_end - job.frame_start + 1}</span>
                  </span>
                </div>

                <ProgressBar
                  current={job.completed_frames}
                  total={job.frame_end - job.frame_start + 1}
                  height={8}
                  showStats={false}
                />
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
        </div>

        <div className="glass overflow-hidden border-white/5 shadow-2xl bg-black/40 p-6">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-white/[0.02] text-[10px] uppercase tracking-[0.2em] text-white/25">
                  <th className="px-8 py-5 font-black text-left border-b border-white/5">{t('job_details.th_node')}</th>
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
                        <div className="flex items-center">
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
                        <div className="flex flex-col items-center gap-2">
                          <StatusBadge status={task.status} />
                          {task.status === 'FAILED' && task.logs && (
                            <button
                              onClick={() => setSelectedTaskLogs({ logs: task.logs!, id: task.id })}
                              className="flex items-center gap-1.5 text-[9px] font-black uppercase tracking-widest text-rose-500/60 hover:text-rose-500 transition-colors"
                            >
                              <FileText size={12} />
                              {t('job_details.show_logs') || 'Show Logs'}
                            </button>
                          )}
                        </div>
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

      <AnimatePresence>
        {selectedTaskLogs && (
          <LogModal
            logs={selectedTaskLogs.logs}
            taskID={selectedTaskLogs.id}
            onClose={() => setSelectedTaskLogs(null)}
          />
        )}
      </AnimatePresence>
    </motion.div>
  );
};

export default JobDetailsPage;
