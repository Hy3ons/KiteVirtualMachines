import React from 'react';
import { Helmet } from 'react-helmet-async';

interface SEOProps {
  title: string;
  description?: string;
  url?: string;
}

export const SEO: React.FC<SEOProps> = ({ title, description, url }) => {
  const defaultDescription = "Kubernetes 기반의 안전하고 빠른 가상머신 프로비저닝 플랫폼, Kite.";
  const siteUrl = "https://kite.anacnu.com"; // Replace with actual production domain when deployed

  return (
    <Helmet>
      <title>{title}</title>
      <meta name="description" content={description || defaultDescription} />
      <meta property="og:title" content={title} />
      <meta property="og:description" content={description || defaultDescription} />
      <meta property="og:url" content={url ? `${siteUrl}${url}` : siteUrl} />
      <meta name="twitter:title" content={title} />
      <meta name="twitter:description" content={description || defaultDescription} />
    </Helmet>
  );
};
