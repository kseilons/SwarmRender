import React from 'react';
import { motion } from 'framer-motion';

interface ProgressBarProps {
  current: number;
  total: number;
  label?: string;
  showStats?: boolean;
  color?: string;
  height?: number;
}

const ProgressBar: React.FC<ProgressBarProps> = ({ 
  current, 
  total, 
  label, 
  showStats = true,
  color = 'var(--accent-primary)',
  height = 6
}) => {
  const percentage = Math.min(Math.max((current / total) * 100, 0), 100);

  return (
    <div className="progress-container">
      {label && (
        <div className="flex justify-between items-center mb-1.5 px-0.5">
          <span className="text-[10px] uppercase font-black tracking-widest text-white/40">{label}</span>
          {showStats && (
            <span className="text-[10px] font-mono font-bold text-cyan-400">{Math.round(percentage)}%</span>
          )}
        </div>
      )}
      
      <div className="progress-track" style={{ height }}>
        <motion.div 
          className="progress-fill-animate"
          initial={{ width: 0 }}
          animate={{ width: `${percentage}%` }}
          style={{ 
            backgroundColor: color,
            opacity: 0.9,
            // Use a simpler background since CSS hex with 'dd' is unreliable if color is a var
            background: color.startsWith('#') ? `linear-gradient(90deg, ${color}, ${color}dd)` : color
          }}
          transition={{ duration: 0.8, ease: "circOut" }}
        />
      </div>

      {showStats && !label && (
        <div className="progress-stats">
          <span>{current} / {total} frames</span>
          <span className="text-cyan-400/80">{Math.round(percentage)}%</span>
        </div>
      )}
    </div>
  );
};

export default ProgressBar;
