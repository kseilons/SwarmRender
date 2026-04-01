import React, { useState } from 'react';
import { X, Upload, Check, Loader2, Info, Folder } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { jobService, CreateJobRequest } from '../services/api';
import { useMutation, useQueryClient, useQuery } from '@tanstack/react-query';
import { useTranslation } from '../context/LanguageContext';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

const CreateJobModal: React.FC<Props> = ({ isOpen, onClose }) => {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const [step, setStep] = useState(1);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [fileCount, setFileCount] = useState(0);
  const [blendFiles, setBlendFiles] = useState<{ name: string, s3Path: string, size: number }[]>([]);
  const fileInputRef = React.useRef<HTMLInputElement>(null);

  const [formData, setFormData] = useState<CreateJobRequest>({
    name: '',
    s3_path: '',
    frame_start: 1,
    frame_end: 250,
    worker_count: 2,
    options: {
      engine: 'CYCLES',
      use_cpu: false,
      cycles_device: 'CUDA',
      threads: 0,
      format: 'PNG',
      scene: 'Scene',
      background: true
    }
  });

  const { data: stats } = useQuery({
    queryKey: ['clusterStats'],
    queryFn: jobService.getStats,
    enabled: isOpen
  });

  const mutation = useMutation({
    mutationFn: jobService.createJob,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
      setStep(3);
      setTimeout(() => {
        onClose();
        setStep(1);
        setFormData({ ...formData, name: '', s3_path: '' });
      }, 2000);
    }
  });

  const onFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    setIsUploading(true);
    setUploadProgress(0);
    setFileCount(files.length);

    let uploadedCount = 0;
    setBlendFiles([]);
    setFormData(prev => ({ ...prev, s3_path: '' }));

    try {
      const projectId = self.crypto.randomUUID();
      let largestFile = { s3Path: '', size: 0 };
      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const relativePath = file.webkitRelativePath || file.name;
        
        // Skip some system files
        if (relativePath.includes('.DS_Store') || relativePath.includes('__MACOSX')) {
          uploadedCount++;
          continue;
        }

        // 1. Get signed URL with project_id
        const { upload_url, s3_path } = await jobService.getUploadUrl(relativePath, projectId);
        
        // 2. Upload to S3
        await jobService.uploadFileToS3(upload_url, file);

        // Keep track of all .blend files found
        if (file.name.endsWith('.blend')) {
          setBlendFiles(prev => [...prev, { name: relativePath, s3Path: s3_path, size: file.size }]);
          if (file.size > largestFile.size) {
            largestFile = { s3Path: s3_path, size: file.size };
          }
        }

        uploadedCount++;
        setUploadProgress(Math.round((uploadedCount / files.length) * 100));
      }

      // Automatically select the largest file by default
      if (largestFile.s3Path) {
        setFormData(prev => ({ ...prev, s3_path: largestFile.s3Path }));
      }

      // If no main file detected or multiple, the last 's3_path' updated in the loop 
      // will be the one. But ideally we want the .blend file's path.
      // The backend 's3_path' from /upload-url already includes the project-specific uuid prefix.
      
    } catch (error) {
      console.error('Upload failed:', error);
      alert('Upload failed. Please try again.');
    } finally {
      setIsUploading(false);
    }
  };

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate(formData);
  };

  if (!isOpen) return null;

  return (
    <div className="modal-overlay">
      <motion.div 
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        className="absolute inset-0"
      />
      
      <motion.div 
        initial={{ scale: 0.9, opacity: 0, y: 20 }}
        animate={{ scale: 1, opacity: 1, y: 0 }}
        className="modal-content"
      >
        <div className="modal-header">
          <h2 className="text-xl font-bold">{t('create_modal.title')}</h2>
          <button onClick={onClose} className="p-1 hover:bg-white/5 rounded-lg transition-colors border-none bg-transparent cursor-pointer text-white">
            <X size={20} />
          </button>
        </div>

        <div className="modal-body">
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.form 
                key="step1"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="flex flex-col gap-6"
                onSubmit={(e) => { e.preventDefault(); setStep(2); }}
              >
                <div className="flex flex-col gap-4">
                  <div className="form-group">
                    <label className="form-label">{t('create_modal.field_name')}</label>
                    <input 
                      type="text" required
                      className="form-input"
                      placeholder="Mars Cinematic Sequence"
                      value={formData.name}
                      onChange={e => setFormData({...formData, name: e.target.value})}
                    />
                  </div>

                  <div className="form-group">
                    <label className="form-label">{t('create_modal.field_file')}</label>
                    <div 
                      onClick={handleUploadClick}
                      className="upload-area"
                      style={formData.s3_path ? {borderColor: 'rgba(16, 185, 129, 0.5)', background: 'rgba(16, 185, 129, 0.05)'} : {}}
                    >
                      <input 
                        type="file" 
                        ref={fileInputRef}
                        className="hidden"
                        onChange={onFileSelect}
                        {...({ webkitdirectory: "", directory: "" } as any)}
                      />
                      {isUploading ? (
                        <div className="w-full">
                           <div className="flex justify-between text-xs font-mono mb-1">
                              <span className="text-cyan-400">Uploading {fileCount} files...</span>
                              <span className="text-white/40">{uploadProgress}%</span>
                           </div>
                           <div className="progress-bar">
                              <div className="progress-fill" style={{ width: `${uploadProgress}%` }}></div>
                           </div>
                        </div>
                      ) : formData.s3_path ? (
                        <>
                          <div className="w-12 h-12 bg-emerald-500/20 rounded-full flex items-center justify-center text-emerald-400" style={{background: 'rgba(16, 185, 129, 0.2)', color: '#10b981', width: '48px', height: '48px', borderRadius: '50%'}}><Check size={24} /></div>
                          <span className="text-sm font-medium" style={{color: '#10b981'}}>{t('create_modal.upload_complete')}</span>
                          <span className="text-xs text-white/30 truncate w-full text-center font-mono">{formData.s3_path.split('/').pop()}</span>
                        </>
                      ) : (
                        <>
                          <div className="flex gap-4 mb-2">
                             <Upload size={24} style={{opacity: 0.3}} />
                             <Folder size={24} style={{opacity: 0.3}} />
                          </div>
                          <span className="text-sm font-medium">{t('create_modal.upload_instruction')}</span>
                          <span className="text-[10px] text-white/20 uppercase tracking-tighter mt-1">Single folder with all project assets</span>
                        </>
                      )}
                    </div>
                  </div>

                  {blendFiles.length > 1 && (
                    <motion.div 
                      initial={{ opacity: 0, y: -10 }}
                      animate={{ opacity: 1, y: 0 }}
                      className="form-group"
                    >
                      <label className="form-label">{t('create_modal.field_main_file')}</label>
                      <select 
                        className="form-input"
                        value={formData.s3_path}
                        onChange={e => setFormData({...formData, s3_path: e.target.value})}
                      >
                        <option value="" disabled>{t('common.unassigned')}</option>
                        {blendFiles.map((file, idx) => (
                          <option key={idx} value={file.s3Path}>
                            {file.name}
                          </option>
                        ))}
                      </select>
                    </motion.div>
                  )}
                </div>

                <div className="flex justify-end pt-4">
                   <button 
                    disabled={!formData.name || !formData.s3_path}
                    className="btn-primary"
                    style={{width: 'auto', padding: '12px 32px'}}
                   >
                     {t('create_modal.continue')} <span style={{marginLeft: '8px'}}>→</span>
                   </button>
                </div>
              </motion.form>
            )}

            {step === 2 && (
              <motion.form 
                key="step2"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="flex flex-col gap-6"
                onSubmit={handleSubmit}
              >
                  <div className="grid sm:grid-cols-2 gap-4">
                    <div className="form-group">
                       <label className="form-label">{t('create_modal.start_frame')}</label>
                       <input 
                        type="number"
                        className="form-input"
                        value={formData.frame_start}
                        onChange={e => setFormData({...formData, frame_start: parseInt(e.target.value)})}
                       />
                    </div>
                    <div className="form-group">
                       <label className="form-label">{t('create_modal.end_frame')}</label>
                       <input 
                        type="number"
                        className="form-input"
                        value={formData.frame_end}
                        onChange={e => setFormData({...formData, frame_end: parseInt(e.target.value)})}
                       />
                    </div>
                  </div>

                  <div className="form-group">
                    <div className="flex justify-between items-center mb-1">
                       <label className="form-label">{t('create_modal.compute_nodes')}</label>
                       <span className="text-xs font-mono text-cyan-400">{formData.worker_count} / {stats?.total_workers || '2'} {t('create_modal.slots')}</span>
                    </div>
                    <input 
                      type="range" min="1" max={stats?.total_workers || 2}
                      style={{width: '100%', cursor: 'pointer', accentColor: '#00d2ff'}}
                      value={formData.worker_count}
                      onChange={e => setFormData({...formData, worker_count: parseInt(e.target.value)})}
                    />
                  </div>

                  <div className="pt-4 border-t border-white/5">
                    <h3 className="text-xs font-bold uppercase tracking-widest text-white/40 mb-4">{t('create_modal.advanced_options')}</h3>
                    
                    <div className="grid sm:grid-cols-2 gap-4">
                      <div className="form-group">
                        <label className="form-label">{t('create_modal.field_engine')}</label>
                        <select 
                          className="form-input"
                          value={formData.options.engine}
                          onChange={e => setFormData({
                            ...formData, 
                            options: { ...formData.options, engine: e.target.value }
                          })}
                        >
                          <option value="CYCLES">CYCLES</option>
                          <option value="EEVEE">EEVEE</option>
                        </select>
                      </div>

                      <div className="form-group">
                         <label className="form-label">{t('create_modal.field_format')}</label>
                         <select 
                          className="form-input"
                          value={formData.options.format}
                          onChange={e => setFormData({
                            ...formData, 
                            options: { ...formData.options, format: e.target.value }
                          })}
                        >
                          <option value="PNG">PNG</option>
                          <option value="JPEG">JPEG</option>
                          <option value="EXR">OPEN_EXR</option>
                        </select>
                      </div>

                      <div className="form-group">
                        <label className="form-label">{t('create_modal.field_scene')}</label>
                        <input 
                          type="text"
                          className="form-input"
                          value={formData.options.scene}
                          onChange={e => setFormData({
                            ...formData, 
                            options: { ...formData.options, scene: e.target.value }
                          })}
                        />
                      </div>

                      <div className="form-group">
                        <label className="form-label">{t('create_modal.field_threads')}</label>
                        <input 
                          type="number"
                          className="form-input"
                          min="0" max="12"
                          value={formData.options.threads}
                          onChange={e => {
                            let val = parseInt(e.target.value) || 0;
                            if (val > 12) val = 12;
                            setFormData({
                              ...formData, 
                              options: { ...formData.options, threads: val }
                            });
                          }}
                        />
                        <p className="text-[10px] text-white/20 mt-1 uppercase tracking-wider">Max 12 threads</p>
                      </div>

                      <div className="form-group">
                        <label className="form-label">{t('create_modal.field_use_cpu')}</label>
                        <div className="flex items-center gap-2 mt-2">
                          <input 
                            type="checkbox"
                            id="use_cpu"
                            checked={formData.options.use_cpu}
                            onChange={e => setFormData({
                              ...formData, 
                              options: { ...formData.options, use_cpu: e.target.checked }
                            })}
                          />
                          <label htmlFor="use_cpu" className="text-sm cursor-pointer">{t('create_modal.field_use_cpu')}</label>
                        </div>
                      </div>

                      {formData.options.engine === 'CYCLES' && !formData.options.use_cpu && (
                        <div className="form-group">
                          <label className="form-label">{t('create_modal.field_device')}</label>
                          <select 
                            className="form-input"
                            value={formData.options.cycles_device}
                            onChange={e => setFormData({
                              ...formData, 
                              options: { ...formData.options, cycles_device: e.target.value }
                            })}
                          >
                            <option value="CUDA">CUDA</option>
                            <option value="OPTIX">OPTIX</option>
                            <option value="OPENCL">OPENCL</option>
                          </select>
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="flex gap-3 pt-6">
                    <button 
                      type="button" onClick={() => setStep(1)}
                      className="btn-secondary"
                      style={{flex: 1, background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)', color: 'white', borderRadius: '12px', cursor: 'pointer', height: '42px', fontWeight: 600, fontSize: '14px'}}
                    >
                      {t('create_modal.go_back')}
                    </button>
                    <button 
                      type="submit"
                      disabled={mutation.isPending}
                      className="btn-primary"
                      style={{flex: 2, height: '42px'}}
                    >
                      {mutation.isPending ? <Loader2 className="animate-spin" size={20} /> : t('create_modal.initialize')}
                    </button>
                  </div>
              </motion.form>
            )}

            {step === 3 && (
              <motion.div 
                key="step3"
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                className="flex flex-col items-center gap-4 py-12"
              >
                <div className="w-16 h-16 rounded-full flex items-center justify-center text-emerald-400" style={{background: 'rgba(16, 185, 129, 0.2)', width: '64px', height: '64px', borderRadius: '50%'}}><Check size={32} /></div>
                <h3 className="text-2xl font-bold">{t('create_modal.step_3_title')}</h3>
                <p className="text-white/40 text-sm">{t('create_modal.notifying_workers')}</p>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
        
        <div className="modal-footer">
           <Info className="text-cyan-500" size={18} style={{flexShrink: 0}} />
           <p className="text-xs text-white/40 leading-relaxed">
             {t('create_modal.pack_textures_info')}
           </p>
        </div>
      </motion.div>
    </div>
  );
};

export default CreateJobModal;
