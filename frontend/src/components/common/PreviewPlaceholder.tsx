import React from 'react';

interface PreviewPlaceholderProps {
  name: string;
  size?: 'sm' | 'lg';
}

const PreviewPlaceholder: React.FC<PreviewPlaceholderProps> = ({ name, size = 'sm' }) => {
  const initials = name.substring(0, 2).toUpperCase();

  // Pick a gradient based on name hash for consistency
  const gradients = [
    'linear-gradient(135deg, rgba(6, 182, 212, 0.2), rgba(59, 130, 246, 0.2))',
    'linear-gradient(135deg, rgba(168, 85, 247, 0.2), rgba(236, 72, 153, 0.2))',
    'linear-gradient(135deg, rgba(16, 185, 129, 0.2), rgba(20, 184, 166, 0.2))',
    'linear-gradient(135deg, rgba(249, 115, 22, 0.2), rgba(244, 63, 94, 0.2))',
  ];
  
  const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  const gradient = gradients[hash % gradients.length];

  const circleSize = size === 'lg' ? 180 : 100;
  const fontSize = size === 'lg' ? 60 : 30;

  return (
    <div 
      className="w-full h-full flex flex-col items-center justify-center relative overflow-hidden" 
      style={{ 
        backgroundColor: '#111', 
        background: 'radial-gradient(circle at center, #1a1a1a 0%, #0a0a0a 100%)',
        width: '100%',
        height: '100%'
      }}
    >
      {/* Dynamic Background Gradient */}
      <div className="absolute inset-0 opacity-30" style={{ background: gradient }} />
      
      {/* Content: Visible Circle with Large Initials */}
      <div className="relative" style={{ zIndex: 10 }}>
        <div 
          className="rounded-full flex items-center justify-center font-bold tracking-tighter text-white"
          style={{ 
            width: `${circleSize}px`, 
            height: `${circleSize}px`, 
            fontSize: `${fontSize}px`,
            border: '2px solid rgba(255, 255, 255, 0.4)',
            boxSizing: 'border-box',
            backgroundColor: 'rgba(255, 255, 255, 0.08)',
            backdropFilter: 'blur(10px)',
            WebkitBackdropFilter: 'blur(10px)',
            boxShadow: '0 8px 32px rgba(0, 0, 0, 0.3)',
            display: 'flex'
          }}
        >
          {initials}
        </div>
      </div>

      {/* Subtle Shimmer Effect */}
      <div className="absolute inset-0 pointer-events-none">
        <div 
          style={{ 
            width: '100%',
            height: '100%',
            background: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.05), transparent)',
            position: 'absolute',
            top: 0,
            left: '-100%',
            animation: 'shimmer-placeholder 3s infinite'
          }} 
        />
        <style dangerouslySetInnerHTML={{ __html: `
          @keyframes shimmer-placeholder {
            100% { left: 100%; }
          }
        `}} />
      </div>
    </div>
  );
};

export default PreviewPlaceholder;
