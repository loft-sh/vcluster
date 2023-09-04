import React from 'react';
import styles from './pro-label.css';
const CustomLabel = ({children, color, href}) => (
    <a href={href} style={{color}} className='proFeatureLabel'> {children}
       </a>
);
export default CustomLabel;