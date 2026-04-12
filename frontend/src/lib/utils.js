export const IDR = (n) =>
  'Rp ' + Math.round(n).toLocaleString('id-ID');

export const fmtDate = (s) =>
  new Date(s).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });

export const fmtDateTime = (s) =>
  new Date(s).toLocaleString('id-ID', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit'
  });

export const STATUS_LABEL = {
  paid: 'Lunas',
  pending: 'Menunggu Bayar',
  script_executed: 'Script Dieksekusi',
  cancelled: 'Dibatalkan'
};

export const PAY_LABEL = {
  bank: 'Transfer Bank',
  dana: 'DANA',
  gopay: 'GoPay',
  ovo: 'OVO',
  qris: 'QRIS',
  crypto: 'Cryptocurrency'
};

export const PRODUCT_ICONS = [
  '📦','🌐','📚','🎮','🎨','⚡','💻','🖥️','📱','🔑',
  '🎵','🎬','📊','🔧','🛡️','📸','🎯','🚀','💡','🔥'
];
