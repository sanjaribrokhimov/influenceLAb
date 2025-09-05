// SEO Templates for Dynamic Content
// Этот файл содержит шаблоны для автоматической генерации SEO meta-тегов

const SEOTemplates = {
    // Шаблоны для блога
    blog: {
        // Meta title шаблоны
        title: {
            ru: (title) => `${title} - Блог Influence Lab | Инфлюенсер-маркетинг в Узбекистане`,
            uz: (title) => `${title} - Influence Lab Blogi | O'zbekistonda influencer marketing`,
            en: (title) => `${title} - Influence Lab Blog | Influencer Marketing in Uzbekistan`
        },
        
        // Meta description шаблоны
        description: {
            ru: (description, title) => `${description.substring(0, 120)}... Читайте больше о ${title.toLowerCase()} в блоге Influence Lab - ведущего агентства инфлюенсер-маркетинга в Узбекистане.`,
            uz: (description, title) => `${description.substring(0, 120)}... ${title.toLowerCase()} haqida ko'proq o'qing Influence Lab blogida - O'zbekistondagi yetakchi influencer marketing agentligi.`,
            en: (description, title) => `${description.substring(0, 120)}... Read more about ${title.toLowerCase()} on Influence Lab blog - leading influencer marketing agency in Uzbekistan.`
        },
        
        // Open Graph title шаблоны
        ogTitle: {
            ru: (title) => `${title} | Influence Lab Blog`,
            uz: (title) => `${title} | Influence Lab Blogi`,
            en: (title) => `${title} | Influence Lab Blog`
        },
        
        // Open Graph description шаблоны
        ogDescription: {
            ru: (description) => `${description.substring(0, 150)}...`,
            uz: (description) => `${description.substring(0, 150)}...`,
            en: (description) => `${description.substring(0, 150)}...`
        },
        
        // Twitter title шаблоны
        twitterTitle: {
            ru: (title) => `${title} - Influence Lab`,
            uz: (title) => `${title} - Influence Lab`,
            en: (title) => `${title} - Influence Lab`
        },
        
        // Twitter description шаблоны
        twitterDescription: {
            ru: (description) => `${description.substring(0, 100)}...`,
            uz: (description) => `${description.substring(0, 100)}...`,
            en: (description) => `${description.substring(0, 100)}...`
        }
    },
    
    // Шаблоны для проектов
    project: {
        // Meta title шаблоны
        title: {
            ru: (title) => `${title} - Проект Influence Lab | Кейсы инфлюенсер-маркетинга`,
            uz: (title) => `${title} - Influence Lab Loyihasi | Influencer marketing keyslari`,
            en: (title) => `${title} - Influence Lab Project | Influencer Marketing Cases`
        },
        
        // Meta description шаблоны
        description: {
            ru: (description, title) => `${description.substring(0, 120)}... Смотрите кейс проекта "${title}" от Influence Lab - успешные примеры инфлюенсер-маркетинга в Узбекистане.`,
            uz: (description, title) => `${description.substring(0, 120)}... "${title}" loyihasi keysini ko'ring Influence Lab'dan - O'zbekistonda muvaffaqiyatli influencer marketing misollari.`,
            en: (description, title) => `${description.substring(0, 120)}... See case study of "${title}" project by Influence Lab - successful influencer marketing examples in Uzbekistan.`
        },
        
        // Open Graph title шаблоны
        ogTitle: {
            ru: (title) => `${title} | Influence Lab Project`,
            uz: (title) => `${title} | Influence Lab Loyihasi`,
            en: (title) => `${title} | Influence Lab Project`
        },
        
        // Open Graph description шаблоны
        ogDescription: {
            ru: (description) => `${description.substring(0, 150)}...`,
            uz: (description) => `${description.substring(0, 150)}...`,
            en: (description) => `${description.substring(0, 150)}...`
        },
        
        // Twitter title шаблоны
        twitterTitle: {
            ru: (title) => `${title} - Influence Lab`,
            uz: (title) => `${title} - Influence Lab`,
            en: (title) => `${title} - Influence Lab`
        },
        
        // Twitter description шаблоны
        twitterDescription: {
            ru: (description) => `${description.substring(0, 100)}...`,
            uz: (description) => `${description.substring(0, 100)}...`,
            en: (description) => `${description.substring(0, 100)}...`
        }
    },
    
    // Шаблоны для LED экранов
    led: {
        // Meta title шаблоны
        title: {
            ru: (title) => `${title} - LED экран в Ташкенте | Influence Lab`,
            uz: (title) => `${title} - Toshkentda LED ekran | Influence Lab`,
            en: (title) => `${title} - LED Screen in Tashkent | Influence Lab`
        },
        
        // Meta description шаблоны
        description: {
            ru: (description, title, location) => `${description.substring(0, 120)}... LED экран "${title}" в ${location || 'Ташкенте'} от Influence Lab. Быстрое размещение рекламы на LED экранах.`,
            uz: (description, title, location) => `${description.substring(0, 120)}... "${title}" LED ekrani ${location || 'Toshkentda'} Influence Lab'dan. LED ekranlarda tez reklama joylashtirish.`,
            en: (description, title, location) => `${description.substring(0, 120)}... LED screen "${title}" in ${location || 'Tashkent'} by Influence Lab. Fast LED screen advertising placement.`
        },
        
        // Open Graph title шаблоны
        ogTitle: {
            ru: (title) => `${title} | LED экран Influence Lab`,
            uz: (title) => `${title} | Influence Lab LED ekran`,
            en: (title) => `${title} | Influence Lab LED Screen`
        },
        
        // Open Graph description шаблоны
        ogDescription: {
            ru: (description) => `${description.substring(0, 150)}...`,
            uz: (description) => `${description.substring(0, 150)}...`,
            en: (description) => `${description.substring(0, 150)}...`
        },
        
        // Twitter title шаблоны
        twitterTitle: {
            ru: (title) => `${title} - Influence Lab`,
            uz: (title) => `${title} - Influence Lab`,
            en: (title) => `${title} - Influence Lab`
        },
        
        // Twitter description шаблоны
        twitterDescription: {
            ru: (description) => `${description.substring(0, 100)}...`,
            uz: (description) => `${description.substring(0, 100)}...`,
            en: (description) => `${description.substring(0, 100)}...`
        }
    }
};

// Функция для генерации SEO meta-тегов
function generateSEOMeta(content, type, lang = 'ru') {
    const templates = SEOTemplates[type];
    if (!templates) return {};
    
    const title = content.title || content.title_ru || '';
    const description = content.description || content.description_ru || '';
    const location = content.location || '';
    
    return {
        title: templates.title[lang](title),
        description: templates.description[lang](description, title, location),
        ogTitle: templates.ogTitle[lang](title),
        ogDescription: templates.ogDescription[lang](description),
        twitterTitle: templates.twitterTitle[lang](title),
        twitterDescription: templates.twitterDescription[lang](description)
    };
}

// Функция для генерации структурированных данных
function generateStructuredData(content, type, lang = 'ru') {
    const baseUrl = 'https://influencelab.uz';
    const title = content.title || content.title_ru || '';
    const description = content.description || content.description_ru || '';
    const image = content.img || content.images?.[0] || '';
    
    let structuredData = {
        "@context": "https://schema.org",
        "@type": type === 'blog' ? "BlogPosting" : type === 'project' ? "CreativeWork" : "Product",
        "headline": title,
        "description": description,
        "url": `${baseUrl}/${type}/${content.id}`,
        "datePublished": new Date().toISOString(),
        "dateModified": new Date().toISOString(),
        "author": {
            "@type": "Organization",
            "name": "Influence Lab",
            "url": baseUrl
        },
        "publisher": {
            "@type": "Organization",
            "name": "Influence Lab",
            "logo": {
                "@type": "ImageObject",
                "url": `${baseUrl}/img/logo.png`
            }
        }
    };
    
    if (image) {
        structuredData.image = {
            "@type": "ImageObject",
            "url": image.startsWith('http') ? image : `${baseUrl}${image}`
        };
    }
    
    if (type === 'led' && content.location) {
        structuredData["@type"] = "Product";
        structuredData.brand = {
            "@type": "Brand",
            "name": "Influence Lab"
        };
        structuredData.offers = {
            "@type": "Offer",
            "availability": "https://schema.org/InStock",
            "priceCurrency": "UZS"
        };
    }
    
    return structuredData;
}

// Экспорт для использования в других файлах
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { SEOTemplates, generateSEOMeta, generateStructuredData };
}
