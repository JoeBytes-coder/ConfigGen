import type { GlobalThemeOverrides } from 'naive-ui'

/**
 * ConfigGen 前端主题
 * 主色 #7DAEE1（柔和的云原生蓝），配套浅色背景与同色系描边，
 * 整体呈现干净、克制的开发者工具气质。
 */
export const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#7DAEE1',
    primaryColorHover: '#93BEE8',
    primaryColorPressed: '#6698CF',
    primaryColorSuppl: '#7DAEE1',

    bodyColor: '#F5F8FC',
    cardColor: '#FFFFFF',
    modalColor: '#FFFFFF',
    popoverColor: '#FFFFFF',

    borderColor: '#DFE9F5',
    dividerColor: '#E6EEF8',

    textColorBase: '#2C3E5D',
    textColor1: '#2C3E5D',
    textColor2: '#44557A',
    textColor3: '#8296B3',

    borderRadius: '8px',
    fontWeightStrong: '600',
  },
}
