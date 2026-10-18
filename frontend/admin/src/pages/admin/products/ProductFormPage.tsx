import { useState, useEffect, useMemo, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  ArrowLeftIcon,
  PlusIcon,
  TrashIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import {
  adminGetProduct,
  createProduct,
  updateProduct,
} from '../../../api/products';
import { listCategories } from '../../../api/categories';
import LoadingSpinner from '../../../components/common/LoadingSpinner';
import ErrorMessage from '../../../components/common/ErrorMessage';
import type {
  CreateProductRequest,
  UpdateProductRequest,
  CreateVariantRequest,
} from '../../../types';

interface VariantForm {
  sku: string;
  price: string;
  compare_at_price: string;
  cost_price: string;
  weight_g: string;
  options: { key: string; value: string }[];
  images: { url: string; alt: string }[];
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-+|-+$/g, '');
}

function createEmptyVariant(): VariantForm {
  return {
    sku: '',
    price: '',
    compare_at_price: '',
    cost_price: '',
    weight_g: '',
    options: [],
    images: [],
  };
}

export default function ProductFormPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [autoSlug, setAutoSlug] = useState(true);
  const [description, setDescription] = useState('');
  const [shortDescription, setShortDescription] = useState('');
  const [brand, setBrand] = useState('');
  const [categoryId, setCategoryId] = useState('');
  const [tagsInput, setTagsInput] = useState('');
  const [sortOrder, setSortOrder] = useState('0');
  const [status, setStatus] = useState('ACTIVE');
  const [unit, setUnit] = useState('piece');
  const [specs, setSpecs] = useState<{ key: string; value: string }[]>([]);
  const [images, setImages] = useState<{ url: string; alt: string }[]>([{ url: '', alt: '' }]);
  const [variants, setVariants] = useState<VariantForm[]>([createEmptyVariant()]);

  // Fetch categories for dropdown
  const categoriesQuery = useQuery({
    queryKey: ['categories'],
    queryFn: listCategories,
  });

  // Fetch product in edit mode
  const productQuery = useQuery({
    queryKey: ['admin', 'product', id],
    queryFn: () => adminGetProduct(id!),
    enabled: isEditing,
  });

  // Populate form when product data loads in edit mode
  useEffect(() => {
    if (isEditing && productQuery.data) {
      const p = productQuery.data;
      setName(p.name);
      setSlug(p.slug);
      setAutoSlug(false);
      setDescription(p.description);
      setShortDescription(p.short_description);
      setBrand(p.brand);
      setCategoryId(p.category_id);
      setTagsInput(p.tags.join(', '));
      setSortOrder(String(p.sort_order));
      setStatus(p.status);
      setUnit(p.unit || 'piece');
      setSpecs(Object.entries(p.specs ?? {}).map(([key, value]) => ({ key, value })));
      setImages(
        p.images.length > 0
          ? p.images.map((img) => ({ url: img.url, alt: img.alt || '' }))
          : [{ url: '', alt: '' }],
      );
      if (p.variants.length > 0) {
        setVariants(
          p.variants.map((v) => ({
            sku: v.sku,
            price: String(v.price),
            compare_at_price: v.compare_at_price ? String(v.compare_at_price) : '',
            cost_price: String(v.cost_price),
            weight_g: v.weight_g ? String(v.weight_g) : '',
            options: Object.entries(v.options ?? {}).map(([key, value]) => ({
              key,
              value,
            })),
            images: v.images.map((img) => ({ url: img.url, alt: img.alt || '' })),
          })),
        );
      }
    }
  }, [isEditing, productQuery.data]);

  // Auto-generate slug from name
  useEffect(() => {
    if (autoSlug && name) {
      setSlug(slugify(name));
    }
  }, [name, autoSlug]);

  const categories = useMemo(
    () => categoriesQuery.data?.data ?? [],
    [categoriesQuery.data],
  );

  const createMutation = useMutation({
    mutationFn: createProduct,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'products'] });
      navigate('/admin/products');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id: productId, data }: { id: string; data: UpdateProductRequest }) =>
      updateProduct(productId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'products'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'product', id] });
      navigate('/admin/products');
    },
  });

  const isSubmitting = createMutation.isPending || updateMutation.isPending;
  const mutationError = createMutation.error || updateMutation.error;

  // Variant management handlers
  const addVariant = useCallback(() => {
    setVariants((prev) => [...prev, createEmptyVariant()]);
  }, []);

  const removeVariant = useCallback((index: number) => {
    setVariants((prev) => {
      if (prev.length <= 1) return prev;
      return prev.filter((_, i) => i !== index);
    });
  }, []);

  const updateVariant = useCallback(
    (index: number, field: keyof VariantForm, value: string | { key: string; value: string }[]) => {
      setVariants((prev) =>
        prev.map((v, i) => (i === index ? { ...v, [field]: value } : v)),
      );
    },
    [],
  );

  const addVariantAttribute = useCallback((variantIndex: number) => {
    setVariants((prev) =>
      prev.map((v, i) =>
        i === variantIndex
          ? { ...v, options: [...v.options, { key: '', value: '' }] }
          : v,
      ),
    );
  }, []);

  const removeVariantAttribute = useCallback((variantIndex: number, attrIndex: number) => {
    setVariants((prev) =>
      prev.map((v, i) =>
        i === variantIndex
          ? { ...v, options: v.options.filter((_, ai) => ai !== attrIndex) }
          : v,
      ),
    );
  }, []);

  const updateVariantAttribute = useCallback(
    (variantIndex: number, attrIndex: number, field: 'key' | 'value', val: string) => {
      setVariants((prev) =>
        prev.map((v, i) =>
          i === variantIndex
            ? {
                ...v,
                options: v.options.map((a, ai) =>
                  ai === attrIndex ? { ...a, [field]: val } : a,
                ),
              }
            : v,
        ),
      );
    },
    [],
  );

  const addVariantImage = useCallback((variantIndex: number) => {
    setVariants((prev) =>
      prev.map((v, i) =>
        i === variantIndex
          ? { ...v, images: [...v.images, { url: '', alt: '' }] }
          : v,
      ),
    );
  }, []);

  const removeVariantImage = useCallback((variantIndex: number, imgIndex: number) => {
    setVariants((prev) =>
      prev.map((v, i) =>
        i === variantIndex
          ? { ...v, images: v.images.filter((_, ii) => ii !== imgIndex) }
          : v,
      ),
    );
  }, []);

  const updateVariantImage = useCallback(
    (variantIndex: number, imgIndex: number, field: 'url' | 'alt', val: string) => {
      setVariants((prev) =>
        prev.map((v, i) =>
          i === variantIndex
            ? {
                ...v,
                images: v.images.map((img, ii) =>
                  ii === imgIndex ? { ...img, [field]: val } : img,
                ),
              }
            : v,
        ),
      );
    },
    [],
  );

  const buildVariantsPayload = (): CreateVariantRequest[] => {
    return variants.map((v) => {
      const opts: Record<string, string> = {};
      for (const o of v.options) {
        if (o.key.trim()) {
          opts[o.key.trim()] = o.value.trim();
        }
      }
      return {
        sku: v.sku.trim(),
        // Money is whole BDT taka; weight is grams.
        price: parseInt(v.price, 10) || 0,
        compare_at_price: v.compare_at_price ? parseInt(v.compare_at_price, 10) : undefined,
        cost_price: parseInt(v.cost_price, 10) || 0,
        weight_g: v.weight_g ? parseInt(v.weight_g, 10) : 0,
        requires_shipping: true,
        options: Object.keys(opts).length > 0 ? opts : undefined,
        images: v.images
          .filter((img) => img.url.trim())
          .map((img, idx) => ({
            url: img.url.trim(),
            alt: img.alt.trim() || undefined,
            sort_order: idx,
          })),
      };
    });
  };

  const buildProductExtras = () => {
    const specMap: Record<string, string> = {};
    for (const sp of specs) {
      if (sp.key.trim()) specMap[sp.key.trim()] = sp.value.trim();
    }
    return {
      status,
      unit: unit.trim() || 'piece',
      specs: specMap,
      images: images
        .filter((img) => img.url.trim())
        .map((img, idx) => ({ url: img.url.trim(), alt: img.alt.trim() || name.trim(), sort_order: idx })),
    };
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const tags = tagsInput
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean);

    const variantPayload = buildVariantsPayload();

    if (isEditing) {
      const data: UpdateProductRequest = {
        name: name.trim(),
        slug: slug.trim(),
        description: description.trim() || undefined,
        short_description: shortDescription.trim() || undefined,
        brand: brand.trim() || undefined,
        category_id: categoryId || undefined,
        tags: tags.length > 0 ? tags : undefined,
        variants: variantPayload,
        sort_order: parseInt(sortOrder, 10) || 0,
        ...buildProductExtras(),
      };
      updateMutation.mutate({ id: id!, data });
    } else {
      const data: CreateProductRequest = {
        name: name.trim(),
        slug: slug.trim(),
        description: description.trim() || undefined,
        short_description: shortDescription.trim() || undefined,
        brand: brand.trim() || undefined,
        category_id: categoryId,
        tags: tags.length > 0 ? tags : undefined,
        variants: variantPayload,
        sort_order: parseInt(sortOrder, 10) || 0,
        ...buildProductExtras(),
      };
      createMutation.mutate(data);
    }
  };

  // Loading state for edit mode
  if (isEditing && productQuery.isLoading) {
    return <LoadingSpinner />;
  }

  if (isEditing && productQuery.isError) {
    return (
      <ErrorMessage
        message={productQuery.error?.message || 'Failed to load product'}
        onRetry={() => productQuery.refetch()}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link
          to="/admin/products"
          className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 transition-colors"
        >
          <ArrowLeftIcon className="h-4 w-4" />
          Back to Products
        </Link>
      </div>

      <div>
        <h1 className="text-2xl font-bold text-gray-900">
          {isEditing ? 'Edit Product' : 'Create New Product'}
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          {isEditing
            ? 'Update product information and variants'
            : 'Fill in the details below to create a new product'}
        </p>
      </div>

      {/* Error */}
      {mutationError && (
        <ErrorMessage
          message={mutationError.message || 'Something went wrong'}
        />
      )}

      <form onSubmit={handleSubmit} className="space-y-8">
        {/* Basic Information */}
        <div className="card space-y-5">
          <h2 className="text-lg font-semibold text-gray-900">Basic Information</h2>

          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <div>
              <label htmlFor="name" className="label-field">
                Name <span className="text-red-500">*</span>
              </label>
              <input
                id="name"
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Product name"
                className="input-field"
              />
            </div>

            <div>
              <label htmlFor="slug" className="label-field">
                Slug
              </label>
              <div className="flex gap-2">
                <input
                  id="slug"
                  type="text"
                  required
                  value={slug}
                  onChange={(e) => {
                    setSlug(e.target.value);
                    setAutoSlug(false);
                  }}
                  placeholder="product-slug"
                  className="input-field"
                />
                <button
                  type="button"
                  onClick={() => {
                    setAutoSlug(true);
                    setSlug(slugify(name));
                  }}
                  className="btn-secondary btn-sm whitespace-nowrap"
                >
                  Auto
                </button>
              </div>
            </div>
          </div>

          <div>
            <label htmlFor="shortDescription" className="label-field">
              Short Description
            </label>
            <input
              id="shortDescription"
              type="text"
              value={shortDescription}
              onChange={(e) => setShortDescription(e.target.value)}
              placeholder="A brief description"
              className="input-field"
            />
          </div>

          <div>
            <label htmlFor="description" className="label-field">
              Description
            </label>
            <textarea
              id="description"
              rows={5}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Full product description..."
              className="input-field"
            />
          </div>

          <div className="grid grid-cols-1 gap-5 sm:grid-cols-3">
            <div>
              <label htmlFor="brand" className="label-field">
                Brand
              </label>
              <input
                id="brand"
                type="text"
                value={brand}
                onChange={(e) => setBrand(e.target.value)}
                placeholder="Brand name"
                className="input-field"
              />
            </div>

            <div>
              <label htmlFor="category" className="label-field">
                Category <span className="text-red-500">*</span>
              </label>
              <select
                id="category"
                required
                value={categoryId}
                onChange={(e) => setCategoryId(e.target.value)}
                className="input-field"
              >
                <option value="">Select a category</option>
                {categories.map((cat) => (
                  <option key={cat.id} value={cat.id}>
                    {cat.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label htmlFor="sortOrder" className="label-field">
                Sort Order
              </label>
              <input
                id="sortOrder"
                type="number"
                value={sortOrder}
                onChange={(e) => setSortOrder(e.target.value)}
                placeholder="0"
                className="input-field"
              />
            </div>
          </div>

          <div>
            <label htmlFor="tags" className="label-field">
              Tags
            </label>
            <input
              id="tags"
              type="text"
              value={tagsInput}
              onChange={(e) => setTagsInput(e.target.value)}
              placeholder="tag1, tag2, tag3 (comma-separated)"
              className="input-field"
            />
          </div>

          <div className="grid grid-cols-1 gap-5 sm:grid-cols-3">
            <div>
              <label htmlFor="status" className="label-field">Status</label>
              <select id="status" value={status} onChange={(e) => setStatus(e.target.value)} className="input-field">
                <option value="ACTIVE">Active (visible in store)</option>
                <option value="DRAFT">Draft (hidden)</option>
                <option value="ARCHIVED">Archived (hidden)</option>
              </select>
            </div>
            <div>
              <label htmlFor="unit" className="label-field">Sold by (unit)</label>
              <input id="unit" list="unit-options" value={unit} onChange={(e) => setUnit(e.target.value)} placeholder="piece, kg, litre, dozen" className="input-field" />
              <datalist id="unit-options">
                {['piece', 'kg', 'g', 'litre', 'dozen', 'pack', 'set', 'pair', 'kit'].map((u) => (<option key={u} value={u} />))}
              </datalist>
            </div>
            <div>
              <label className="label-field">Currency</label>
              <input value="BDT (৳)" disabled className="input-field bg-gray-50" />
            </div>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="label-field mb-0">Product images (first one is the main image)</label>
              <button type="button" onClick={() => setImages((p) => [...p, { url: '', alt: '' }])} className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50">
                <PlusIcon className="h-3 w-3" /> Add image
              </button>
            </div>
            <div className="space-y-2">
              {images.map((img, i) => (
                <div key={i} className="flex items-center gap-2">
                  <input value={img.url} onChange={(e) => setImages((p) => p.map((x, j) => (j === i ? { ...x, url: e.target.value } : x)))} placeholder="Image URL (https://...)" className="input-field flex-1" />
                  <input value={img.alt} onChange={(e) => setImages((p) => p.map((x, j) => (j === i ? { ...x, alt: e.target.value } : x)))} placeholder="Alt text" className="input-field w-40" />
                  <button type="button" onClick={() => setImages((p) => (p.length > 1 ? p.filter((_, j) => j !== i) : [{ url: '', alt: '' }]))} className="rounded p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50">
                    <XMarkIcon className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="label-field mb-0">Specifications (shown on the product page)</label>
              <button type="button" onClick={() => setSpecs((p) => [...p, { key: '', value: '' }])} className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50">
                <PlusIcon className="h-3 w-3" /> Add spec
              </button>
            </div>
            {specs.length === 0 ? (
              <p className="text-xs text-gray-400 py-1">e.g. Origin: Rajshahi, Grade: A, Warranty: 1 year</p>
            ) : (
              <div className="space-y-2">
                {specs.map((sp, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <input value={sp.key} onChange={(e) => setSpecs((p) => p.map((x, j) => (j === i ? { ...x, key: e.target.value } : x)))} placeholder="Name (e.g. Origin)" className="input-field flex-1" />
                    <input value={sp.value} onChange={(e) => setSpecs((p) => p.map((x, j) => (j === i ? { ...x, value: e.target.value } : x)))} placeholder="Value (e.g. Rajshahi)" className="input-field flex-1" />
                    <button type="button" onClick={() => setSpecs((p) => p.filter((_, j) => j !== i))} className="rounded p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50">
                      <XMarkIcon className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Variants Section */}
        <div className="card space-y-5">
          <div className="flex items-center justify-between">
            <div><h2 className="text-lg font-semibold text-gray-900">Variants</h2><p className="text-xs text-gray-500">One variant per sellable pack or size (e.g. 1kg, 5kg). Stock is added from Purchases, not here.</p></div>
            <button
              type="button"
              onClick={addVariant}
              className="btn-secondary btn-sm inline-flex items-center gap-1.5"
            >
              <PlusIcon className="h-4 w-4" />
              Add Variant
            </button>
          </div>

          {variants.length === 0 && (
            <div className="text-center py-8 text-sm text-gray-400">
              No variants added. Click "Add Variant" to create one.
            </div>
          )}

          {variants.map((variant, vIdx) => (
            <div
              key={vIdx}
              className="rounded-lg border border-gray-200 p-4 space-y-4"
            >
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-gray-700">
                  Variant {vIdx + 1}
                </h3>
                <button
                  type="button"
                  onClick={() => removeVariant(vIdx)}
                  disabled={variants.length <= 1}
                  className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  <TrashIcon className="h-3.5 w-3.5" />
                  Remove
                </button>
              </div>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <div>
                  <label className="label-field">SKU <span className="text-red-500">*</span></label>
                  <input
                    type="text"
                    required
                    value={variant.sku}
                    onChange={(e) => updateVariant(vIdx, 'sku', e.target.value)}
                    placeholder="SKU-001"
                    className="input-field"
                  />
                </div>
                <div>
                  <label className="label-field">Selling price (৳) <span className="text-red-500">*</span></label>
                  <input
                    type="number"
                    required
                    step="1"
                    min="0"
                    value={variant.price}
                    onChange={(e) => updateVariant(vIdx, 'price', e.target.value)}
                    placeholder="0"
                    className="input-field"
                  />
                </div>
                <div>
                  <label className="label-field">Old price (৳, strikethrough)</label>
                  <input
                    type="number"
                    step="1"
                    min="0"
                    value={variant.compare_at_price}
                    onChange={(e) => updateVariant(vIdx, 'compare_at_price', e.target.value)}
                    placeholder="0"
                    className="input-field"
                  />
                </div>
                <div>
                  <label className="label-field">Cost price (৳) <span className="text-red-500">*</span></label>
                  <input
                    type="number"
                    required
                    step="1"
                    min="0"
                    value={variant.cost_price}
                    onChange={(e) => updateVariant(vIdx, 'cost_price', e.target.value)}
                    placeholder="0"
                    className="input-field"
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="label-field">Weight (grams)</label>
                  <input
                    type="number"
                    step="1"
                    min="0"
                    value={variant.weight_g}
                    onChange={(e) => updateVariant(vIdx, 'weight_g', e.target.value)}
                    placeholder="e.g. 500"
                    className="input-field"
                  />
                </div>
              </div>

              {/* Attributes */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="label-field mb-0">Options (what the buyer picks: size, color, pack)</label>
                  <button
                    type="button"
                    onClick={() => addVariantAttribute(vIdx)}
                    className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 transition-colors"
                  >
                    <PlusIcon className="h-3 w-3" />
                    Add Option
                  </button>
                </div>
                {variant.options.length === 0 ? (
                  <p className="text-xs text-gray-400 py-1">No options</p>
                ) : (
                  <div className="space-y-2">
                    {variant.options.map((attr, aIdx) => (
                      <div key={aIdx} className="flex items-center gap-2">
                        <input
                          type="text"
                          value={attr.key}
                          onChange={(e) =>
                            updateVariantAttribute(vIdx, aIdx, 'key', e.target.value)
                          }
                          placeholder="Key (e.g. Color)"
                          className="input-field flex-1"
                        />
                        <input
                          type="text"
                          value={attr.value}
                          onChange={(e) =>
                            updateVariantAttribute(vIdx, aIdx, 'value', e.target.value)
                          }
                          placeholder="Value (e.g. Red)"
                          className="input-field flex-1"
                        />
                        <button
                          type="button"
                          onClick={() => removeVariantAttribute(vIdx, aIdx)}
                          className="rounded p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors"
                        >
                          <XMarkIcon className="h-4 w-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Images */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="label-field mb-0">Images</label>
                  <button
                    type="button"
                    onClick={() => addVariantImage(vIdx)}
                    className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 transition-colors"
                  >
                    <PlusIcon className="h-3 w-3" />
                    Add Image
                  </button>
                </div>
                {variant.images.length === 0 ? (
                  <p className="text-xs text-gray-400 py-1">No images</p>
                ) : (
                  <div className="space-y-2">
                    {variant.images.map((img, iIdx) => (
                      <div key={iIdx} className="flex items-center gap-2">
                        <input
                          type="text"
                          value={img.url}
                          onChange={(e) =>
                            updateVariantImage(vIdx, iIdx, 'url', e.target.value)
                          }
                          placeholder="Image URL"
                          className="input-field flex-1"
                        />
                        <input
                          type="text"
                          value={img.alt}
                          onChange={(e) =>
                            updateVariantImage(vIdx, iIdx, 'alt', e.target.value)
                          }
                          placeholder="Alt text"
                          className="input-field w-40"
                        />
                        <button
                          type="button"
                          onClick={() => removeVariantImage(vIdx, iIdx)}
                          className="rounded p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors"
                        >
                          <XMarkIcon className="h-4 w-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3">
          <Link to="/admin/products" className="btn-secondary">
            Cancel
          </Link>
          <button
            type="submit"
            disabled={isSubmitting}
            className="btn-primary inline-flex items-center gap-2"
          >
            {isSubmitting && (
              <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            )}
            {isEditing ? 'Update Product' : 'Create Product'}
          </button>
        </div>
      </form>
    </div>
  );
}
