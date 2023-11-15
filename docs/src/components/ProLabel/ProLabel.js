import React from "react";
import "./pro-label.css";

const CustomLabel = ({ children, color, href }) => (
  <a href={href} style={{ color }} className="proFeatureLabel">
    {" "}
    {children}
  </a>
);
export default CustomLabel;
