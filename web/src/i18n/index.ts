import { messages } from './messages';

export function getT(lang: string) {
  return (path: string, params: Record<string, any> = {}) => {
    const keys = path.split('.');
    let obj = (messages as any)[lang];
    for (const k of keys) {
      if (!obj) return path;
      obj = obj[k];
    }
    let str = obj || path;
    if (typeof str === 'string') {
      Object.keys(params).forEach(key => {
        str = str.replace(`{${key}}`, params[key]);
      });
    }
    return str;
  };
}

export const messages_data = messages;
