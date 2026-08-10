/**
 * Custom ESLint rule for ensuring icon-only Button components have aria-label or aria-labelledby.
 * Targets design-system Button atom with size="icon" | "icon-xs" | "icon-sm" | "icon-lg"
 * and no visible text child (excluding sr-only).
 */

export default {
  meta: {
    type: "problem",
    docs: {
      description:
        "Icon-only Button components must have aria-label or aria-labelledby (WCAG 2.1 Level A)",
      category: "Accessibility",
      recommended: true,
    },
    messages: {
      missingAriaLabel:
        "Icon-only Button (size='icon*') must have aria-label or aria-labelledby. Screen readers will announce 'button' without a label.",
    },
  },

  create(context) {
    return {
      JSXOpeningElement(node) {
        // Only check Button elements
        if (node.name.name !== "Button") return;

        // Get the size prop value
        const sizeAttr = node.attributes.find(
          (attr) =>
            attr.type === "JSXAttribute" &&
            attr.name.name === "size" &&
            attr.value?.type === "Literal"
        );
        if (!sizeAttr) return;

        const sizeValue = sizeAttr.value.value;
        const isIconSize = /^icon(-xs|-sm|-lg)?$/.test(sizeValue);
        if (!isIconSize) return;

        // Check for aria-label or aria-labelledby
        const hasAriaLabel = node.attributes.some(
          (attr) =>
            attr.type === "JSXAttribute" &&
            (attr.name.name === "aria-label" || attr.name.name === "aria-labelledby")
        );
        if (hasAriaLabel) return;

        // Button is a JSXOpeningElement. To check children, we need to look at the parent JSXElement
        // which wraps this opening element.
        const sourceCode = context.sourceCode;
        const parent = sourceCode.getAncestors(node).find((n) => n.type === "JSXElement");
        if (!parent || parent.openingElement !== node) return;

        const hasVisibleText = parent.children.some((child) => {
          if (child.type === "JSXText") {
            return child.value.trim().length > 0;
          }
          if (child.type === "JSXElement") {
            // Exclude sr-only elements
            const classAttr = child.openingElement.attributes.find(
              (attr) =>
                attr.type === "JSXAttribute" &&
                attr.name.name === "className" &&
                attr.value?.type === "Literal"
            );
            if (classAttr && classAttr.value.value.includes("sr-only")) {
              return false;
            }
            return true; // Other JSX elements are considered visible
          }
          return false;
        });

        if (!hasVisibleText) {
          context.report({
            node,
            messageId: "missingAriaLabel",
          });
        }
      },
    };
  },
};
